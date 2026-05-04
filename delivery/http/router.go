package http

import (
	"flag-gen/application/usecase"
	"flag-gen/delivery/http/handler"
	"flag-gen/infrastructure/di"

	"github.com/gin-gonic/gin"
)

func RegisterV1Routes(r *gin.Engine, app *di.ApplicationState) {
	v1 := r.Group("/v1")

	projectsHandler := handler.NewProjectsHandler(
		usecase.NewListProjectsUsecase(app.Projects),
		usecase.NewCreateProjectUsecase(app.Projects),
	)
	envHandler := handler.NewEnvironmentsHandler(
		usecase.NewListEnvironmentsUsecase(app.Environments),
		usecase.NewCreateEnvironmentUsecase(app.Environments),
	)
	segmentsHandler := handler.NewSegmentsHandler(
		usecase.NewListSegmentsUsecase(app.Segments),
		usecase.NewCreateSegmentUsecase(app.Segments),
		usecase.NewUpdateSegmentUsecase(app.Segments),
		usecase.NewDeleteSegmentUsecase(app.Segments),
	)
	flagsHandler := handler.NewFlagsHandler(
		usecase.NewListFlagsUsecase(app.Flags),
		usecase.NewGetFlagUsecase(app.Flags),
		usecase.NewCreateFlagUsecase(app.Flags),
		usecase.NewUpdateFlagUsecase(app.Flags),
		usecase.NewDeleteFlagUsecase(app.Flags),
		app.Idempotency,
	)
	evalHandler := handler.NewEvaluateHandler(
		usecase.NewEvaluateFlagsUsecase(app.ScopedFlags),
	)

	v1.GET("/projects", projectsHandler.List)
	v1.POST("/projects", projectsHandler.Create)

	v1.GET("/projects/:projectId/environments", envHandler.List)
	v1.POST("/projects/:projectId/environments", envHandler.Create)

	v1.GET("/projects/:projectId/segments", segmentsHandler.List)
	v1.POST("/projects/:projectId/segments", segmentsHandler.Create)
	v1.PATCH("/projects/:projectId/segments/:segmentId", segmentsHandler.Update)
	v1.DELETE("/projects/:projectId/segments/:segmentId", segmentsHandler.Delete)

	v1.GET("/projects/:projectId/environments/:environmentKey/flags", flagsHandler.List)
	v1.POST("/projects/:projectId/environments/:environmentKey/flags", flagsHandler.Create)
	v1.GET("/projects/:projectId/environments/:environmentKey/flags/:flagKey", flagsHandler.GetByKey)
	v1.PATCH("/projects/:projectId/environments/:environmentKey/flags/:flagId", flagsHandler.Update)
	v1.DELETE("/projects/:projectId/environments/:environmentKey/flags/:flagId", flagsHandler.Delete)

	v1.POST("/projects/:projectId/environments/:environmentKey/flags/evaluate", evalHandler.Evaluate)
}

