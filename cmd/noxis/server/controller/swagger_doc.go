package controller

// Documentation-only annotation carriers. The change-submit route takes either
// a SQL change or a MongoDB request; swag cannot express that polymorphism on a
// single operation, so the concrete schema is emitted from here (it has no
// @Router, so it adds no path) and the operation references `object`. Exported
// so the unused linter leaves it alone.

// SwaggerMongoChangeBody documents the MongoDB change-submit body.
// @Param body body mongoChangeBody true "mongodb request"
func SwaggerMongoChangeBody() {}
