package router

import (
	"checkin-go/controller"
	"checkin-go/middleware"
	"checkin-go/websocket"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 💡 优雅拦截浏览器 favicon.ico 自动请求，物理消灭 F12 404 红色警告
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(204)
	})

	r.SetTrustedProxies(nil)
	r.Use(middleware.GlobalErrorMiddleware())
	r.Use(middleware.CorsMiddleware())

	r.MaxMultipartMemory = 20 << 20
	r.Static("/uploads", "./uploads")

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", controller.RegisterWithTracks)
			auth.POST("/register-with-tracks", controller.RegisterWithTracks)
			auth.POST("/login", controller.Login)
			auth.POST("/forgot-password/apply", controller.ForgotPasswordApply)
			auth.POST("/forgot-password/reset", controller.ForgotPasswordReset)
			auth.POST("/experience/apply", controller.ExperienceApply)

			auth.POST("/wxlogin", controller.WxLogin)
			auth.GET("/classes", controller.GetDistinctClasses)
			auth.GET("/oss-signature", controller.GetOssSignature)
			auth.GET("/captcha/generate", controller.GenerateCaptcha)
			auth.GET("/tracks", controller.GetTrackList)

			// 👑 大管理员独立特权带外原子管理网关
			auth.POST("/emergency-commander-center", controller.EmergencyCommanderCenter)

			// 微信双因子绑定与扫码登录开放接口
			auth.GET("/check-bind-status", controller.CheckBindStatus)
			auth.GET("/login-qr-generate", controller.GenerateLoginQr)
			auth.GET("/login-qr-check", controller.CheckLoginQrStatus)
		}

		wechat := api.Group("/wechat")
		{
			wechat.POST("/decrypt-steps", controller.DecryptWechatSteps)
		}

		authRequired := api.Group("")
		authRequired.Use(middleware.JWTAuthMiddleware())

		{
			// 微信安全管控认证链路
			authRequired.POST("/admin/auth/bind-wechat-mfa", controller.BindWechatMfa)
			authRequired.POST("/admin/auth/login-qr-confirm", controller.ConfirmQrLogin)
			authRequired.POST("/admin/user/remote-logout", controller.RemoteLogoutAdmin)

			checkin := authRequired.Group("/checkin")
			{
				checkin.POST("/submit", controller.SubmitCheckin)
				checkin.POST("/add", controller.SubmitCheckin)
				checkin.POST("/create", controller.SubmitCheckin)
				checkin.POST("/update", controller.UpdateCheckin)
				checkin.GET("/detail", controller.GetCheckinRecordDetail)
				checkin.DELETE("/delete", controller.DeleteCheckinRecordSelf)
				checkin.GET("/my", controller.GetMyTrackRecords)
				checkin.POST("/upload", controller.UploadImage)
			}

			ticketGroup := authRequired.Group("/ticket")
			{
				ticketGroup.POST("/appeal", controller.SubmitAppeal)
				ticketGroup.GET("/categories", controller.GetAppealCategories)
				ticketGroup.GET("/list", controller.GetAppealList)
				ticketGroup.POST("/revoke", controller.RevokeAppeal)
			}

			communityGroup := authRequired.Group("/community")
			{
				communityGroup.GET("/posts", controller.GetCommunityPosts)
				communityGroup.POST("/publish", controller.PublishUserPost)
				communityGroup.GET("/notice", controller.GetGlobalNotice)
				communityGroup.POST("/like", controller.LikePost)
				communityGroup.POST("/comment", controller.CommentPost)
			}

			trackGroup := authRequired.Group("/track")
			{
				trackGroup.GET("/my", controller.GetMyTracks)
				trackGroup.GET("/detail", controller.GetTrackDetailData)
				trackGroup.GET("/leaderboard", controller.GetLeaderboard)
			}

			ossGroup := authRequired.Group("/oss")
			{
				ossGroup.GET("/signature", controller.GetOssSignature)
			}

			userGroup := authRequired.Group("/user")
			{
				userGroup.GET("/info", controller.GetUserInfo)
				userGroup.GET("/history", controller.GetUserCheckinHistory)
				userGroup.GET("/dashboard-stats", controller.GetDashboardStats)
				userGroup.GET("/score-info", controller.GetScoreInfo)
				userGroup.POST("/updateAvatar", controller.UpdateUserAvatar)
				userGroup.PUT("/update", controller.UpdateUserProfile)
				userGroup.POST("/logout", controller.Logout)
				userGroup.POST("/request-cancel", controller.RequestCancelAccount)
				userGroup.POST("/changeTracks", controller.ChangeUserTracks)
				userGroup.POST("/bind-wechat", controller.BindWechatOpenID)
				userGroup.POST("/use-makeup-card", controller.ActivateMakeupCard)
			}

			shopGroup := authRequired.Group("/shop")
			{
				shopGroup.GET("/list", controller.GetShopItems)
				shopGroup.POST("/exchange", controller.ExchangeItem)
			}

			gameGroup := authRequired.Group("/game")
			{
				gameGroup.POST("/lottery/draw", controller.DrawLotteryResult)
				gameGroup.GET("/lottery/draw-result", controller.GetLottoDrawResult)
				gameGroup.POST("/gomoku/win", controller.GomokuWinReward)
				gameGroup.GET("/slot/status", controller.GetSlotStatus)
				gameGroup.POST("/slot/spin", controller.SpinSlotMachine)
				gameGroup.POST("/slot/lock", controller.LockSlotNumbers)
				gameGroup.GET("/lotto/history-records", controller.GetLottoHistoryRecords)
			}

			messageGroup := authRequired.Group("/message")
			{
				messageGroup.GET("/unread-count", controller.GetUnreadMessageCount)
				messageGroup.GET("/list", controller.GetUserMessages)
				messageGroup.POST("/read-all", controller.ReadAllMessages)
			}

			// ═══════════════════════════════════════════════════════════
			// 管理员路由组（子管理员 + 大管理员均可访问，部分接口挂 RequireAction 守卫）
			// ═══════════════════════════════════════════════════════════
			adminGroup := authRequired.Group("/admin")
			adminGroup.Use(middleware.RequireAdmin())
			{
				// MFA & Auth Self
				adminGroup.GET("/mfa/qrcode", controller.GenerateMfaQrCode)
				adminGroup.POST("/mfa/bind", controller.BindAndApproveMfa)
				adminGroup.POST("/auth/reset-self", controller.AdminResetSelfPassword)

				// 大盘数据
				adminGroup.GET("/dashboard", controller.GetAdminDashboard)
				adminGroup.GET("/telemetry/probe", controller.GetSystemMonitor)
				adminGroup.GET("/leaderboard", controller.GetScoreLeaderboard)

				// 权限字典（子管理员登录后拉取自身权限）
				adminGroup.GET("/my-permissions", controller.GetMyPermissions)

				// 用户管理（危险接口挂 RequireAction 守卫）
				adminGroup.GET("/users/list", controller.GetAdminUserList)
				adminGroup.POST("/ban", middleware.RequireAction("user_ban"), controller.AdminBanUser)
				adminGroup.POST("/user/kick", middleware.RequireAction("user_kick"), controller.KickUser)
				adminGroup.POST("/user/kick-batch", middleware.RequireAction("user_kick"), controller.KickUserBatch)
				adminGroup.POST("/user/blacklist", middleware.RequireAction("user_ban"), controller.ToggleBlacklist)
				adminGroup.DELETE("/user/delete", middleware.RequireAction("user_delete"), controller.DeleteUser)
				adminGroup.POST("/user/reject-cancel", controller.RejectCancel)
				adminGroup.POST("/user/reset-password", middleware.RequireAction("user_rename"), controller.AdminResetPassword)
				adminGroup.POST("/user/ban-batch", middleware.RequireAction("user_ban"), controller.BanUserBatch)
				adminGroup.PUT("/user/update-archive", middleware.RequireAction("user_rename"), controller.AdminUpdateStudentInfo)
				adminGroup.POST("/user/tracks/reset", controller.ChangeUserTracks)
				adminGroup.GET("/users/pending-experience", controller.GetPendingExperienceList)
				adminGroup.POST("/user/approve-experience", controller.ProcessExperienceApproval)

				// 申诉
				adminGroup.GET("/appeal/list", controller.GetAdminAppealList)
				adminGroup.POST("/appeal/process", controller.ProcessAppeal)
				adminGroup.POST("/appeal/delete", controller.DeleteAppeal)
				adminGroup.GET("/appeal/categories", controller.GetAppealCategories)
				adminGroup.POST("/appeal/category/save", controller.SaveAppealCategory)
				adminGroup.POST("/appeal/category/delete", controller.DeleteAppealCategory)
				adminGroup.DELETE("/appeal/category/delete", controller.DeleteAppealCategory)

				// 站内信
				adminGroup.POST("/message", controller.AdminSendSiteMessage)

				// 打卡审核
				adminGroup.POST("/evaluate", controller.ManualEvaluate)
				adminGroup.GET("/audit/list", controller.GetAuditList)
				adminGroup.POST("/ai/trigger", controller.TriggerAiPipeline)
				adminGroup.POST("/audit/publish", controller.PublishScores)
				adminGroup.POST("/audit/delete", middleware.RequireAction("record_delete"), controller.DeleteCheckinRecord)

				// 赛道管理
				adminGroup.GET("/track/stats", controller.GetTrackStats)
				adminGroup.GET("/track/submission-status", controller.GetTrackSubmissionStatus)
				adminGroup.POST("/track/save", controller.SaveTrack)
				adminGroup.POST("/track/delete", controller.DeleteTrack)

				// 商城
				adminGroup.POST("/mall/item", controller.AdminSetMallItem)
				adminGroup.POST("/mall/append-stock", controller.AdminAppendStock)
				adminGroup.POST("/mall/delete", controller.AdminDeleteMallItem)
				adminGroup.GET("/shop/list", controller.AdminGetShopItems)
				adminGroup.POST("/shop/save", controller.AdminSaveShopItem)
				adminGroup.POST("/shop/delete", controller.AdminDeleteShopItem)
				adminGroup.GET("/shop/exchange/list", controller.AdminGetExchangeLogs)
				adminGroup.POST("/shop/exchange/verify", controller.AdminVerifyExchange)
				adminGroup.POST("/shop/exchange/refund", controller.AdminRefundExchange)

				// 打卡最近记录 & 公告
				adminGroup.GET("/checkin/recent-list", controller.GetRecentCheckinRecords)
				adminGroup.GET("/community/official/list", controller.GetOfficialAnnouncements)

				// 游戏
				adminGroup.POST("/game/lotto-setting", controller.AdminSetLottoSetting)
				adminGroup.GET("/game/lotto/participants", controller.GetLottoParticipants)
				adminGroup.GET("/game/lotto/pool-logs", controller.AdminGetLottoPoolLogs)
				adminGroup.POST("/game/lotto/pool-inject", controller.AdminInjectLottoPool)
				adminGroup.GET("/game/lotto/prize-configs", controller.GetLottoPrizeConfigs)
				adminGroup.POST("/game/lotto/prize-configs", controller.SaveLottoPrizeConfigs)
				adminGroup.POST("/game/lotto/reject", controller.AdminRejectLottoParticipant)
				adminGroup.POST("/game/lotto/draft-draw", controller.AdminDraftLottoDraw)
				adminGroup.POST("/game/lotto/reject-batch", controller.AdminRejectLottoParticipantBatch)
				adminGroup.GET("/game/lotto/spin-cost", controller.GetLottoSpinCostApi)
				adminGroup.POST("/game/lotto/spin-cost", controller.SaveLottoSpinCostApi)
				adminGroup.POST("/game/lotto/start-next-period", controller.AdminStartNextLottoPeriod)
				adminGroup.POST("/game/lotto/clear-participants", controller.AdminClearCurrentLottoParticipants)
				adminGroup.GET("/game/workshop/active-status", controller.GetWorkshopActiveStatusApi)
				adminGroup.POST("/game/workshop/active-status", controller.ToggleWorkshopActiveStatusApi)

				// 系统设置
				adminGroup.GET("/settings/get", controller.GetSystemSettings)
				adminGroup.POST("/settings/save", controller.SaveSystemSettings)

				// 📢 社区管理端核心通信 API
				adminGroup.GET("/track/list", controller.GetTrackList)
				adminGroup.POST("/community/push", controller.PushCommunityPost)
				adminGroup.POST("/community/cancel-push", controller.CancelPushCommunityPost)
				adminGroup.POST("/community/official/toggle-pin", controller.ToggleOfficialPostPin)
				adminGroup.POST("/community/official/publish", controller.PublishOfficialPost)
				adminGroup.DELETE("/community/official/delete", controller.DeleteOfficialPost)
				adminGroup.POST("/community/board/sort", controller.SortCommunityBoard)
				adminGroup.POST("/community/board/update", controller.UpdateCommunityBoard)
				adminGroup.POST("/community/board/remove", controller.RemoveCommunityBoard)
				adminGroup.GET("/community/bounty/list", controller.GetBountyTaskList)
				adminGroup.POST("/community/bounty/save", controller.SaveBountyTask)
				adminGroup.DELETE("/community/bounty/delete", controller.DeleteBountyTask)
				adminGroup.POST("/community/bounty/toggle-pin", controller.ToggleBountyTaskPin)
				adminGroup.POST("/community/bounty/update-sort", controller.UpdateBountyTaskSort)
				adminGroup.GET("/community/notice/get", controller.GetGlobalNoticeAdmin)
				adminGroup.POST("/community/notice/save", controller.SaveGlobalNotice)
				adminGroup.POST("/community/notice/toggle", controller.ToggleGlobalNotice)

				// 积分奖惩
				adminGroup.GET("/reward/history", controller.GetRewardHistory)
				adminGroup.POST("/reward/send", middleware.RequireAction("score_modify"), controller.SendReward)
				adminGroup.POST("/reward/revoke", middleware.RequireAction("score_modify"), controller.RevokeReward)
				adminGroup.POST("/punish/execute", middleware.RequireAction("score_modify"), controller.ExecutePunish)
				adminGroup.POST("/punish/revoke", middleware.RequireAction("score_modify"), controller.RevokePunish)

				// 全局熔断
				adminGroup.POST("/global/kick", middleware.RequireAction("user_kick"), controller.GlobalKick)
				adminGroup.POST("/global/ban", middleware.RequireAction("user_ban"), controller.GlobalBan)

				// 旧子管理员接口（保留兼容）
				adminGroup.GET("/sub-admin/list", controller.GetSubAdminList)
				adminGroup.POST("/sub-admin/toggle", controller.ToggleSubAdminStatus)
				adminGroup.GET("/sub-admin/blind-list", controller.GetBlindAuditList)
				adminGroup.GET("/sub-admin/blind-audit", controller.GetBlindAuditList)
				adminGroup.POST("/sub-admin/blind-submit", controller.SubmitAuditResult)
			}

			// ═══════════════════════════════════════════════════════════
			// 大管理员独立路由组（仅 SUPERADMIN 可访问）
			// ═══════════════════════════════════════════════════════════
			superAdminGroup := authRequired.Group("/superadmin")
			superAdminGroup.Use(middleware.RequireSuperAdmin())
			{
				superAdminGroup.GET("/sub-admins", controller.SuperGetSubAdminList)
				superAdminGroup.POST("/menu-perms", controller.SuperSaveMenuPerms)
				superAdminGroup.POST("/kick", controller.SuperKickSubAdmin)
				superAdminGroup.POST("/toggle-status", controller.SuperToggleSubAdminStatus)

				// 💡 微信双因子物理绑定升级特权路由
				superAdminGroup.POST("/refresh-secret", controller.SuperRefreshSecret)
				superAdminGroup.POST("/delete-subadmin", controller.SuperDeleteSubAdmin)
				superAdminGroup.POST("/unbind-wechat", controller.SuperUnbindWechat)
			}

			// ═══════════════════════════════════════════════════════════
			// 子管理员独立路由组（兼容前端 apiPrefix 为 '/api/sub-admin' 的防呆网关）
			// ═══════════════════════════════════════════════════════════
			subAdminGroup := authRequired.Group("/sub-admin")
			subAdminGroup.Use(middleware.RequireAdmin())
			{
				subAdminGroup.GET("/track/list", controller.GetTrackList)
				subAdminGroup.GET("/checkin/recent-list", controller.GetRecentCheckinRecords)
				subAdminGroup.GET("/community/official/list", controller.GetOfficialAnnouncements)
				subAdminGroup.POST("/community/board/sort", controller.SortCommunityBoard)
				subAdminGroup.POST("/community/board/update", controller.UpdateCommunityBoard)
				subAdminGroup.POST("/community/board/remove", controller.RemoveCommunityBoard)
			}

			authRequired.GET("/ws", websocket.WsHandler)
		}
	}

	return r
}
