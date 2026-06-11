package dao

import (
	"checkin-go/global"
	"checkin-go/models"
)

func FindUserByStudentID(studentID string) (*models.User, error) {
	var user models.User
	err := global.DB.Where("student_id = ?", studentID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func CreateUser(user *models.User) error {
	return global.DB.Create(user).Error
}

func UpdateUserDevice(userID int64, deviceID string) error {
	return global.DB.Model(&models.User{}).Where("id = ?", userID).Update("current_device_id", deviceID).Error
}
