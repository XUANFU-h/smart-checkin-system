package service

import (
	"checkin-go/global"
	"checkin-go/models"
	"errors"
)

// 拉取审批列表
func GetAuthApprovalList(status string) ([]map[string]interface{}, error) {
	var list []map[string]interface{}

	err := global.DB.Table("auth_approval").
		Select("auth_approval.id as approvalId, auth_approval.apply_type as applyType, auth_approval.create_time as createTime, user.real_name as name, user.class_name as className").
		Joins("left join user on user.id = auth_approval.user_id").
		Where("auth_approval.status = ?", status).
		Order("auth_approval.create_time desc").
		Scan(&list).Error

	return list, err
}

// 管理员处理审批
func ProcessAuthApproval(approvalID int64, action string) error {
	var approval models.AuthApproval
	if err := global.DB.First(&approval, approvalID).Error; err != nil {
		return errors.New("找不到该审批工单")
	}
	if approval.Status != 0 {
		return errors.New("该工单已被处理")
	}

	newStatus := 2 // REJECT
	if action == "APPROVE" {
		newStatus = 1
	}
	global.DB.Model(&approval).Update("status", newStatus)

	if newStatus == 1 {
		if approval.ApplyType == "RESET_PWD" {
			// 将密码重置为默认的 "123456"
			hashedPwd, _ := hashPassword("123456")
			global.DB.Model(&models.User{}).Where("id = ?", approval.UserID).Update("password", hashedPwd)
		} else if approval.ApplyType == "EXPERIENCE" {
			// 激活体验账户
			global.DB.Model(&models.User{}).Where("id = ?", approval.UserID).Update("user_type", 3)
		}

		// TODO: 下一轮在这里注入 WebSocket 推送逻辑！瞬间通知前端放行！
	}

	return nil
}
