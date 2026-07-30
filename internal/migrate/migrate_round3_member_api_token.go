package migrate

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"git.itopcms.com/jackliu/doc/internal/config"
	"git.itopcms.com/jackliu/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

const round3MemberApiTokenVersion = int64(202607301700)

// MigrationRound3MemberAPIToken creates md_member_api_tokens for MCP HTTP Bearer tokens.
type MigrationRound3MemberAPIToken struct {
	isValid bool
	tables  []string
}

func NewMigrationRound3MemberAPIToken() *MigrationRound3MemberAPIToken {
	return &MigrationRound3MemberAPIToken{}
}

func (m *MigrationRound3MemberAPIToken) Version() int64 { return round3MemberApiTokenVersion }

func (m *MigrationRound3MemberAPIToken) ValidUpdate(version int64) error {
	if m.Version() > version {
		m.isValid = true
		return nil
	}
	m.isValid = false
	return errors.New("The target version is higher than the current version.")
}

func (m *MigrationRound3MemberAPIToken) ValidForBackupTableSchema() error {
	if !m.isValid {
		return errors.New("The current version failed to verify.")
	}
	var err error
	m.tables, err = ExportDatabaseTable()
	return err
}

func (m *MigrationRound3MemberAPIToken) ValidForUpdateTableSchema() error {
	if !m.isValid {
		return errors.New("The current version failed to verify.")
	}
	table := config.GetDatabasePrefix() + "member_api_tokens"
	adapter := strings.ToLower(config.MustGlobal().Database.Adapter)
	o := orm.NewOrm()

	exists, err := tableExists(o, adapter, table)
	if err != nil {
		return err
	}
	if exists {
		logs.Info("table %s already exists, skip create", table)
		return nil
	}

	var ddl string
	switch adapter {
	case "sqlite3":
		ddl = fmt.Sprintf(`CREATE TABLE "%s" (
  "token_id" INTEGER PRIMARY KEY AUTOINCREMENT,
  "member_id" INTEGER NOT NULL,
  "token_hash" VARCHAR(64) NOT NULL UNIQUE,
  "name" VARCHAR(100) NOT NULL DEFAULT '',
  "scopes" VARCHAR(255) NOT NULL DEFAULT 'read',
  "expires_at" DATETIME NULL,
  "last_used_at" DATETIME NULL,
  "last_used_ip" VARCHAR(45) NULL,
  "revoked_at" DATETIME NULL,
  "created_at" DATETIME NOT NULL
)`, table)
	default:
		ddl = fmt.Sprintf("CREATE TABLE `%s` ("+
			"`token_id` INT NOT NULL AUTO_INCREMENT,"+
			"`member_id` INT NOT NULL,"+
			"`token_hash` VARCHAR(64) NOT NULL,"+
			"`name` VARCHAR(100) NOT NULL DEFAULT '',"+
			"`scopes` VARCHAR(255) NOT NULL DEFAULT 'read',"+
			"`expires_at` DATETIME NULL,"+
			"`last_used_at` DATETIME NULL,"+
			"`last_used_ip` VARCHAR(45) NULL,"+
			"`revoked_at` DATETIME NULL,"+
			"`created_at` DATETIME NOT NULL,"+
			"PRIMARY KEY (`token_id`),"+
			"UNIQUE KEY `uk_token_hash` (`token_hash`),"+
			"KEY `idx_member_id` (`member_id`)"+
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4", table)
	}
	if _, err := o.Raw(ddl).Exec(); err != nil {
		return fmt.Errorf("create %s: %w", table, err)
	}
	logs.Info("created table %s", table)
	return nil
}

func tableExists(o orm.Ormer, adapter, table string) (bool, error) {
	switch adapter {
	case "sqlite3":
		var name string
		err := o.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).QueryRow(&name)
		if err == orm.ErrNoRows {
			return false, nil
		}
		return name != "", err
	default:
		var cnt int
		err := o.Raw(
			`SELECT COUNT(1) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`,
			table,
		).QueryRow(&cnt)
		return cnt > 0, err
	}
}

func (m *MigrationRound3MemberAPIToken) MigrationOldTableData() error {
	if !m.isValid {
		return errors.New("The current version failed to verify.")
	}
	return nil
}

func (m *MigrationRound3MemberAPIToken) MigrationNewTableData() error {
	if !m.isValid {
		return errors.New("The current version failed to verify.")
	}
	return nil
}

func (m *MigrationRound3MemberAPIToken) AddMigrationRecord(version int64) error {
	o := orm.NewOrm()
	tables, err := ExportDatabaseTable()
	if err != nil {
		return err
	}
	migration := model.NewMigration()
	migration.Version = version
	migration.Status = "update"
	migration.CreateTime = time.Now()
	migration.Name = fmt.Sprintf("update_%d", version)
	migration.Statements = strings.Join(tables, "\r\n")
	_, err = o.Insert(migration)
	return err
}

func (m *MigrationRound3MemberAPIToken) MigrationCleanup() error { return nil }

func (m *MigrationRound3MemberAPIToken) RollbackMigration() error {
	if !m.isValid {
		return errors.New("The current version failed to verify.")
	}
	table := config.GetDatabasePrefix() + "member_api_tokens"
	_, err := orm.NewOrm().Raw(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)).Exec()
	return err
}
