package must

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
)

/************************************/
/********** Helper Functions ********/
/************************************/
var validate = validator.New()

func Validate(args interface{}) {
	if err := validate.Struct(args); err != nil {
		spew.Dump(args)
		log.WithError(err).Fatal("validation failed")
	}
}

func BeNil(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
