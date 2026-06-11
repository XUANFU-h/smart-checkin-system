package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ⚡ 探针局部高性能缓存与并发控制锁，以防大量高频轮询同步卡满系统资源与线程
var (
	monitorCache      SystemMonitorResp
	lastMonitorTime   time.Time
	monitorCacheMutex sync.Mutex
)

// ==================== 1. 赛道引擎 ====================

// GetMyTracks 获取用户已选赛道
func GetMyTracks(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	var tracks []models.Track
	sql := `SELECT t.* FROM track t INNER JOIN user_track ut ON t.id = ut.track_id WHERE ut.user_id = ?`
	global.DB.Raw(sql, userID).Scan(&tracks)
	common.Success(c, tracks)
}

// GetTrackDetailData 赛道详情页：昨日双榜单 + 个人足迹
func GetTrackDetailData(c *gin.Context) {
	trackID := c.Query("trackId")
	userID := c.MustGet("userId").(int64)

	var user models.User
	global.DB.Select("class_name").First(&user, userID)

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	globalSql := `SELECT u.id as userId, u.nickname as userName, u.class_name as className, SUM(IFNULL(c.ai_score, 0)) as score 
				  FROM checkin_record c JOIN user u ON c.user_id = u.id 
				  WHERE c.track_id = ? AND DATE(c.create_time) = ? 
				  GROUP BY u.id HAVING score > 0 ORDER BY score DESC`

	var globalList []map[string]interface{}
	global.DB.Raw(globalSql, trackID, yesterday).Scan(&globalList)

	var classList []map[string]interface{}
	myGlobalRank, myClassRank, myYesterdayScore := 0, 0, 0

	classCurrentRank := 1
	for i, row := range globalList {
		rowUserID := row["userId"].(int64)
		rowClass := row["className"].(string)

		if rowUserID == userID {
			myGlobalRank = i + 1
			myYesterdayScore = int(row["score"].(int64))
		}

		if rowClass == user.ClassName {
			classList = append(classList, row)
			if rowUserID == userID {
				myClassRank = classCurrentRank
			}
			classCurrentRank++
		}
	}

	var myRecords []models.CheckinRecord
	global.DB.Where("user_id = ? AND track_id = ?", userID, trackID).Order("create_time desc").Find(&myRecords)

	common.Success(c, gin.H{
		"globalList":       globalList,
		"classList":        classList,
		"myGlobalRank":     myGlobalRank,
		"myClassRank":      myClassRank,
		"myYesterdayScore": myYesterdayScore,
		"myRecords":        myRecords,
	})
}

// GetLeaderboard 数据大盘聚合排行榜
func GetLeaderboard(c *gin.Context) {
	leaderboardType := c.DefaultQuery("type", "global")
	userID := c.MustGet("userId").(int64)

	var user models.User
	if err := global.DB.First(&user, userID).Error; err != nil {
		common.Error(c, "用户不存在")
		return
	}

	yesterdayStr := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	var result []map[string]interface{}

	query := global.DB.Table("checkin_record c").
		Select("u.real_name, u.nickname, u.class_name, SUM(c.ai_score) as score").
		Joins("JOIN user u ON c.user_id = u.id").
		Where("c.checkin_date = ? AND c.status = 1 AND u.user_type != 1 AND u.account_status = 0", yesterdayStr)

	if leaderboardType == "class" {
		query = query.Where("u.class_name = ?", user.ClassName)
	} else if leaderboardType != "global" && leaderboardType != "" {
		query = query.Where("c.track_id = ?", leaderboardType)
	}

	err := query.Group("c.user_id").Order("score DESC").Limit(50).Scan(&result).Error
	if err != nil {
		common.Error(c, "拉取昨日数据排行榜失败")
		return
	}
	common.Success(c, result)
}

// ==================== 2. 管理员：数据大盘聚合 ====================

// TrackAttendance 赛道考勤统计
type TrackAttendance struct {
	TrackName  string `json:"trackName"`
	TotalUsers int64  `json:"totalUsers"`
	CheckedIn  int64  `json:"checkedIn"`
}

// GetAdminDashboard 获取大盘仪表盘核心数据 (金融总账 + 赛道考勤)
// 子管理员自动按分派的 track_id 过滤，大管理员返回全局数据
func GetAdminDashboard(c *gin.Context) {
	todayStr := time.Now().Format("2006-01-02")
	data := make(map[string]interface{})

	userID := c.MustGet("userId").(int64)
	userRole, _ := c.Get("userRole")
	roleStr, _ := userRole.(string)

	// 子管理员：先查询其分配的赛道 ID 列表
	var allowedTrackIDs []int64
	if roleStr != "SUPERADMIN" {
		global.DB.Table("sys_admin_track_relation").
			Where("admin_id = ?", userID).
			Pluck("track_id", &allowedTrackIDs)
	}

	// 1. 获取赛道打卡态势数据
	var tracks []models.Track
	trackQuery := global.DB.Where("is_active = ?", true)
	if len(allowedTrackIDs) > 0 {
		trackQuery = trackQuery.Where("id IN ?", allowedTrackIDs)
	} else if roleStr != "SUPERADMIN" && roleStr != "ADMIN" {
		// 无赛道权限的非超管，返回空
		data["trackAttendance"] = []TrackAttendance{}
		data["ledger"] = gin.H{"totalIssuedEmpower": 0, "totalConsumedEmpower": 0, "totalIssuedMoral": 0, "totalInStockPool": 0}
		common.Success(c, data)
		return
	}
	trackQuery.Limit(4).Find(&tracks)

	var attendanceList []TrackAttendance = []TrackAttendance{}
	for _, t := range tracks {
		var totalUsers int64
		global.DB.Model(&models.UserTrack{}).Where("track_id = ?", t.ID).Count(&totalUsers)

		var checkedIn int64
		global.DB.Model(&models.CheckinRecord{}).Where("track_id = ? AND checkin_date = ? AND status = 1", t.ID, todayStr).Count(&checkedIn)

		attendanceList = append(attendanceList, TrackAttendance{
			TrackName:  t.Name,
			TotalUsers: totalUsers,
			CheckedIn:  checkedIn,
		})
	}
	data["trackAttendance"] = attendanceList

	// 2. 金融总账大事务计算（大管理员全局，子管理员按赛道范围过滤）
	// 构造赛道用户子集（用于积分过滤）
	var userIDsInScope []int64
	if len(allowedTrackIDs) > 0 {
		// 子管理员只看分配赛道内的用户
		global.DB.Table("user_track").Where("track_id IN ?", allowedTrackIDs).Pluck("DISTINCT user_id", &userIDsInScope)
	}

	var totalIssuedEmpower int64
	var totalConsumedEmpower int64
	var totalIssuedMoral int64
	var userEmpowerStock int64
	var userMoralStock int64

	empowerIssuedQ := global.DB.Table("score_log").Where("score_type = 'EMPOWER' AND score > 0")
	empowerConsumedQ := global.DB.Table("score_log").Where("score_type = 'EMPOWER' AND score < 0")
	moralQ := global.DB.Table("score_log").Where("score_type = 'MORAL' AND score > 0")
	empowerStockQ := global.DB.Table("user").Where("user_type = 0 AND account_status = 0")
	moralStockQ := global.DB.Table("user").Where("user_type = 0 AND account_status = 0")

	if len(userIDsInScope) > 0 {
		empowerIssuedQ = empowerIssuedQ.Where("user_id IN ?", userIDsInScope)
		empowerConsumedQ = empowerConsumedQ.Where("user_id IN ?", userIDsInScope)
		moralQ = moralQ.Where("user_id IN ?", userIDsInScope)
		empowerStockQ = empowerStockQ.Where("id IN ?", userIDsInScope)
		moralStockQ = moralStockQ.Where("id IN ?", userIDsInScope)
	}

	empowerIssuedQ.Select("COALESCE(SUM(score), 0)").Row().Scan(&totalIssuedEmpower)
	empowerConsumedQ.Select("COALESCE(SUM(ABS(score)), 0)").Row().Scan(&totalConsumedEmpower)
	moralQ.Select("COALESCE(SUM(score), 0)").Row().Scan(&totalIssuedMoral)
	empowerStockQ.Select("COALESCE(SUM(empowerment_score), 0)").Row().Scan(&userEmpowerStock)
	moralStockQ.Select("COALESCE(SUM(moral_score), 0)").Row().Scan(&userMoralStock)
	totalInStockPool := userEmpowerStock + userMoralStock

	data["ledger"] = gin.H{
		"totalIssuedEmpower":   totalIssuedEmpower,
		"totalConsumedEmpower": totalConsumedEmpower,
		"totalIssuedMoral":     totalIssuedMoral,
		"totalInStockPool":     totalInStockPool,
	}

	common.Success(c, data)
}

// SystemMonitorResp 服务器指标响应结构
type SystemMonitorResp struct {
	CpuUsage    float64 `json:"cpuUsage"`    // CPU 负载 %
	MemoryUsage float64 `json:"memoryUsage"` // 内存占用 %
	DiskFree    float64 `json:"diskFree"`    // 磁盘剩余空间 GB
	PingLatency int     `json:"pingLatency"` // 网络心跳延迟 ms
}

// helper for Linux CPU stats
func getCPUSample() (idle, total uint64, err error) {
	contents, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "cpu" {
			var tVal uint64
			var iVal uint64
			for i, field := range fields[1:] {
				val, _ := strconv.ParseUint(field, 10, 64)
				tVal += val
				if i == 3 { // 4th field is idle
					iVal = val
				}
			}
			return iVal, tVal, nil
		}
	}
	return 0, 0, fmt.Errorf("cpu info not found")
}

// GetSystemMonitor 抓取 43.143.215.16 服务器硬件物理探针接口
func GetSystemMonitor(c *gin.Context) {
	monitorCacheMutex.Lock()
	defer monitorCacheMutex.Unlock()

	// 💡 10秒内直接使用内存极速缓存，杜绝高频并发物理探针命令重复调用卡顿
	if time.Since(lastMonitorTime) < 10*time.Second && monitorCache.CpuUsage > 0 {
		common.Success(c, monitorCache)
		return
	}

	resp := SystemMonitorResp{
		CpuUsage:    15.4,
		MemoryUsage: 45.6,
		DiskFree:    128.4,
		PingLatency: 10,
	}

	// 1. CPU Usage Probe (Windows/Linux 真实探针分支)
	if runtime.GOOS == "windows" {
		out, err := exec.Command("wmic", "cpu", "get", "LoadPercentage").Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && line != "LoadPercentage" {
					val, _ := strconv.ParseFloat(line, 64)
					if val > 0 {
						resp.CpuUsage = val
					}
					break
				}
			}
		}
	} else if runtime.GOOS == "linux" {
		idle1, total1, err1 := getCPUSample()
		time.Sleep(50 * time.Millisecond)
		idle2, total2, err2 := getCPUSample()
		if err1 == nil && err2 == nil {
			idleDiff := idle2 - idle1
			totalDiff := total2 - total1
			if totalDiff > 0 {
				resp.CpuUsage = (1.0 - float64(idleDiff)/float64(totalDiff)) * 100
			}
		}
	}

	// 2. Memory Usage Probe (Windows/Linux 真实探针分支)
	if runtime.GOOS == "windows" {
		out, err := exec.Command("wmic", "OS", "get", "FreePhysicalMemory,TotalVisibleMemorySize").Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			var free, total float64
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.Contains(line, "FreePhysicalMemory") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						free, _ = strconv.ParseFloat(parts[0], 64)
						total, _ = strconv.ParseFloat(parts[1], 64)
						if total > 0 {
							resp.MemoryUsage = ((total - free) / total) * 100
						}
					}
					break
				}
			}
		}
	} else if runtime.GOOS == "linux" {
		contents, err := os.ReadFile("/proc/meminfo")
		if err == nil {
			var total, avail float64
			lines := strings.Split(string(contents), "\n")
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if fields[0] == "MemTotal:" {
						total, _ = strconv.ParseFloat(fields[1], 64)
					} else if fields[0] == "MemAvailable:" {
						avail, _ = strconv.ParseFloat(fields[1], 64)
					}
				}
			}
			if total > 0 {
				resp.MemoryUsage = ((total - avail) / total) * 100
			}
		}
	}

	// 3. Disk Free Probe (Windows/Linux 真实探针分支)
	if runtime.GOOS == "windows" {
		out, err := exec.Command("wmic", "logicaldisk", "where", "DeviceID='C:'", "get", "FreeSpace").Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && line != "FreeSpace" {
					val, _ := strconv.ParseFloat(line, 64)
					resp.DiskFree = val / 1024 / 1024 / 1024 // 转换成 GB
					break
				}
			}
		}
	} else if runtime.GOOS == "linux" {
		out, err := exec.Command("df", "-B1", "/").Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) >= 2 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 4 {
					freeBytes, _ := strconv.ParseFloat(fields[3], 64)
					resp.DiskFree = freeBytes / 1024 / 1024 / 1024 // 转换成 GB
				}
			}
		}
	}

	// 4. Ping Latency Probe (Ping 局域网/国内极速心跳，加入 100ms/1s 严苛超时控制)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "1", "-w", "100", "223.5.5.5")
	} else {
		cmd = exec.Command("ping", "-c", "1", "-W", "1", "223.5.5.5")
	}
	start := time.Now()
	if err := cmd.Run(); err == nil {
		resp.PingLatency = int(time.Since(start).Milliseconds())
	} else {
		resp.PingLatency = 99 // 超时降级兜底，不阻断正常运行
	}

	// 💡 更新缓存实体与上次计算时间戳
	monitorCache = resp
	lastMonitorTime = time.Now()

	common.Success(c, resp)
}

// GetScoreLeaderboard 排行榜加固：Top 10 + 10分钟 Redis 缓存期
func GetScoreLeaderboard(c *gin.Context) {
	ctx := c.Request.Context()
	cacheKey := "leaderboard:empowerment:top10"

	// 1. 尝试从 Redis 检索热点缓存
	if global.Redis != nil {
		cachedData, err := global.Redis.Get(ctx, cacheKey).Result()
		if err == nil && cachedData != "" {
			c.Header("X-Cache-Status", "HIT")
			var list []map[string]interface{}
			if json.Unmarshal([]byte(cachedData), &list) == nil {
				common.Success(c, list)
				return
			}
		}
	}

	// 2. 缓存失效，安全查询 MySQL 核心 user 表 (强行 LIMIT 10)
	var list []map[string]interface{} = []map[string]interface{}{}
	err := global.DB.Table("user").
		Select("id, nickname, real_name, class_name, student_id, empowerment_score").
		Where("user_type = 0 AND account_status = 0"). // 过滤非在籍学生与异常状态
		Order("empowerment_score DESC").
		Limit(10).
		Scan(&list).Error

	if err != nil {
		common.Error(c, "获取排行榜失败: "+err.Error())
		return
	}

	// 3. 将结果反向存入 Redis 保持 10 分钟
	if global.Redis != nil && len(list) > 0 {
		jsonData, err := json.Marshal(list)
		if err == nil {
			global.Redis.Set(ctx, cacheKey, string(jsonData), 10*time.Minute)
		}
	}

	c.Header("X-Cache-Status", "MISS")
	common.Success(c, list)
}
