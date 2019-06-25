package operation

import (
	"github.com/jinzhu/gorm"
	"mirrors_status/pkg/db/mysql"
	"mirrors_status/pkg/model/constants"
	"time"
)

type MirrorOperation struct {
	Id            int                             `gorm:"primary_key" json:"id"`
	Index         string                          `json:"index"`
	CreateDate    time.Time                       `json:"create_date"`
	Username      string                          `json:"username"`
	OperationType string                          `json:"operation_type"`
	MirrorId      string                          `json:"mirror_id"`
	Status        constants.MirrorOperationStatus `json:"status"`
	Msg           string                          `json:"msg"`

	Failed   int `json:"failed"`
	Finish   int `json:"finish"`
	Unfinish int `json:"unfinish"`
	Total    int `json:"total"`
	Retry    int `json:"retry"`
}

func (m MirrorOperation) CreateMirrorOperation() error {
	return mysql.NewMySQLClient().Table("mirror_operations").Create(&m).Error
}

func GetOperationsByUsername(username string) (operations []MirrorOperation, err error) {
	err = mysql.NewMySQLClient().Table("mirror_operations").Where("username = ?", username).Scan(&operations).Error
	return
}

func GetOperationsByMirrorIndex(mirror string) (operations []MirrorOperation, err error) {
	err = mysql.NewMySQLClient().Table("mirror_operations").Where("mirror_id = ?", mirror).Scan(&operations).Error
	return
}

type MirrorOperations struct {
	Operations []MirrorOperation
}

func GetOperationsByDateDesc() (operations []MirrorOperation, err error) {
	err = mysql.NewMySQLClient().Table("mirror_operations").Order("create_date DESC").Scan(&operations).Error
	return
}

func DelOperationByUsername(username string) error {
	return mysql.NewMySQLClient().Table("mirror_operations").Where("username = ?", username).Delete(&MirrorOperation{}).Error
}

func GetOperationByIndex(index string) (op MirrorOperation, err error) {
	err = mysql.NewMySQLClient().Table("mirror_operations").Where("`index` = ?", index).Find(&op).Error
	return
}

func SyncMirrorFinishOnce(index string) (err error) {
	err = mysql.NewMySQLClient().Table("mirror_operations").Where("`index` = ?", index).Update("finish", gorm.Expr("finish + ?", 1)).Error
	return
}

func SyncMirrorFailedOnce(index string) (err error) {
	err = mysql.NewMySQLClient().Table("mirror_operations").Where("`index` = ?", index).Update("failed", gorm.Expr("failed + ?", 1)).Error
	return
}

func SyncMirrorUnfinishOnce(index string) (err error) {
	err = mysql.NewMySQLClient().Table("mirror_operations").Where("`index` = ?", index).Update("unfinish", gorm.Expr("unfinish + ?", 1)).Error
	return
}

func SyncMirrorClearRecord(index string) (err error) {
	err = mysql.NewMySQLClient().Table("mirror_operations").Where("`index` = ?", index).
		Update("retry", gorm.Expr("retry + ?", 1)).
		Update("status", constants.STATUS_WAITING).
		Update("failed", 0).
		Update("finish", 0).
		Update("unfinish", 0).
		Update("msg", "").Error
	return
}

func UpdateMirrorTaskMsg(index, msg string) (err error) {
	err = mysql.NewMySQLClient().Table("mirror_operations").Where("`index` = ?", index).Update("msg", msg).Error
	return
}

func UpdateMirrorStatus(index string, status constants.MirrorOperationStatus, msg string) error {
	return mysql.NewMySQLClient().Table("mirror_operations").Where("`index` = ?", index).Update("status", status).Update("msg", msg).Error
}
