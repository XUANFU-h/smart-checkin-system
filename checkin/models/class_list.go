package models

// ClassList 标准班级字典表结构体模型
type ClassList struct {
	ID        int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ClassName string `gorm:"column:class_name;type:varchar(64);not null;uniqueIndex:uk_class_name" json:"class_name"`
}

// TableName 指定 GORM 映射的标准表名
func (ClassList) TableName() string {
	return "class_list"
}
