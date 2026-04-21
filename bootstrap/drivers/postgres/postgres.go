package postgres

import (
	ginbootstrap "github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/bootstrap/internaldialects"
)

func init() {
	ginbootstrap.MustRegisterDialector(internaldialects.Postgres, "postgres", "postgresql")
}
