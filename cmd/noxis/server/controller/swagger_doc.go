package controller

// Documentation-only annotation carriers. The generic stats route answers with
// one of three shapes depending on {engine}, and the change-submit route takes
// either a SQL change or a MongoDB request. swag cannot express that
// polymorphism on a single operation, so the concrete schemas are emitted from
// here (these have no @Router, so they add no paths) and the operations
// reference `object`. Exported so the unused linter leaves them alone.

// SwaggerStatsSQL documents the SQL stats schema.
// @Success 200 {object} sql.Stats
func SwaggerStatsSQL() {}

// SwaggerStatsMemcache documents the memory-cache stats schema.
// @Success 200 {object} memcache.Stats
func SwaggerStatsMemcache() {}

// SwaggerStatsMongo documents the MongoDB stats schema.
// @Success 200 {object} mongodb.Stats
func SwaggerStatsMongo() {}

// SwaggerMongoChangeBody documents the MongoDB change-submit body.
// @Param body body mongoChangeBody true "mongodb request"
func SwaggerMongoChangeBody() {}
