package cli

import (
	"errors"
	"fmt"

	"git.itopcms.com/jackliu/doc/internal/model"
	passutil "git.itopcms.com/jackliu/doc/pkg/password"
	"github.com/spf13/cobra"
)

var (
	passwordAccount string
	passwordValue   string
)

var passwordCmd = &cobra.Command{
	Use:   "password",
	Short: "Change a member password",
	RunE: func(cmd *cobra.Command, args []string) error {
		bootstrapFromFlags()
		return ChangePassword(passwordAccount, passwordValue)
	},
}

func init() {
	passwordCmd.Flags().StringVar(&passwordAccount, "account", "", "user account")
	passwordCmd.Flags().StringVar(&passwordValue, "password", "", "new password")
	_ = passwordCmd.MarkFlagRequired("account")
	_ = passwordCmd.MarkFlagRequired("password")
	rootCmd.AddCommand(passwordCmd)
}

// ChangePassword updates the password for the given account.
func ChangePassword(account, password string) error {
	if account == "" {
		return errors.New("account cannot be empty")
	}
	if password == "" {
		return errors.New("password cannot be empty")
	}
	member, err := model.NewMember().FindByAccount(account)
	if err != nil {
		return fmt.Errorf("failed to change password: %w", err)
	}
	pwd, err := passutil.PasswordHash(password)
	if err != nil {
		return fmt.Errorf("failed to change password: %w", err)
	}
	member.Password = pwd
	if err := member.Update("password"); err != nil {
		return fmt.Errorf("failed to change password: %w", err)
	}
	fmt.Println("Successfully modified.")
	return nil
}
