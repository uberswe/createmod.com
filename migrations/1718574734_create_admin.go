package migrations

import (
	"github.com/joho/godotenv"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		envFile, err := godotenv.Read(".env")
		if err != nil {
			return err
		}

		if envFile["CREATE_ADMIN"] == "true" {
			dao := daos.New(db)

			admin := &models.Admin{}
			admin.Email = "local@createmod.com"
			admin.SetPassword("jfq.utb*jda2abg!WCR")

			return dao.SaveAdmin(admin)
		}
		return nil
	}, func(db dbx.Builder) error {

		dao := daos.New(db)

		admin, _ := dao.FindAdminByEmail("local@createmod.com")
		if admin != nil {
			return dao.DeleteAdmin(admin)
		}

		return nil
	})
}
