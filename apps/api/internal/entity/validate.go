package entity

import "github.com/go-playground/validator/v10"

// validate is the single validator instance for entity structs; tags on the wire models drive the required-field checks.
var validate = validator.New()
