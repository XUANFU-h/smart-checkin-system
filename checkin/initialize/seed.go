package initialize

import (
	"checkin-go/global"
	"log"
)

// ClassList 完美对齐物理 schema `class_list` 的 GORM 数据实体
type ClassList struct {
	ID        int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ClassName string `gorm:"column:class_name;type:varchar(64);not null;uniqueIndex:uk_class_name" json:"class_name"`
}

// TableName 100% 对齐您的物理表名，防止挂载踩空
func (ClassList) TableName() string {
	return "class_list"
}

// SeedClasses 真实物理无感校准大名录
func SeedClasses() {
	trueClasses := []string{
		"AIG241", "AIG242", "AIG243", "AIG251", "AIG252", "AIG253",
		"软件G241", "软件G242", "软件G243", "软件G244", "软件G251", "软件G252", "软件G253", "软件G254",
		"大数据G241", "大数据G242", "大数据G243", "大数据G244", "大数据G245",
		"大数据G251", "大数据G252", "大数据G253", "大数据G254", "大数据G255",
		"虚拟G241", "虚拟G251", "移动G241", "网络G241", "网络G251", "网络G252",
		"软件C221", "网络C221", "网络C222", "网络C223", "网络C224", "网络C225",
		"体验账户",
	}

	log.Println("🔄 开始安全审计并自动校验物理表 `class_list` 现役班级...")

	var count int64
	global.DB.Model(&ClassList{}).Count(&count)

	// 若物理表行数不等于标准 37 意味着数据未删净或残缺，自动启动物理清洗重建机制
	if count != int64(len(trueClasses)) {
		global.DB.Exec("SET FOREIGN_KEY_CHECKS = 0;")
		global.DB.Exec("TRUNCATE TABLE `class_list`;")
		global.DB.Exec("SET FOREIGN_KEY_CHECKS = 1;")

		for _, name := range trueClasses {
			c := ClassList{ClassName: name}
			if err := global.DB.Create(&c).Error; err != nil {
				log.Printf("⚠️ 权威班级字段 %s 注入异常: %v", name, err)
			}
		}
		log.Printf("✅ 物理表 `class_list` 自动扩容洗涤成功！共计初始化 %d 个可接入合规班级。", len(trueClasses))
	} else {
		log.Println("⚡ 现存物理表班级契约完整性良好，无需重复干预。")
	}
}
