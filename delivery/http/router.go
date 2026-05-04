package http

import (
	"flag-gen/application/usecase"
	"flag-gen/delivery/http/handler"
	"flag-gen/delivery/http/middleware"
	"flag-gen/infrastructure/di"

	"github.com/gin-gonic/gin"
)

func RegisterV1Routes(r *gin.Engine, app *di.ApplicationState) {
	v1 := r.Group("/v1")

	systemHandler := handler.NewSystemHandler(app.System)
	v1.GET("/system/status", systemHandler.Status)
	v1.GET("/system/metrics", systemHandler.Metrics)
	v1.POST("/system/install", systemHandler.Install)
	v1.POST("/system/migrate", systemHandler.Migrate)

	listFlags := usecase.NewListFlagsUsecase(app.Flags)

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
		listFlags,
		usecase.NewGetFlagUsecase(app.Flags),
		usecase.NewCreateFlagUsecase(app.Flags, app.SnapshotPointer),
		usecase.NewUpdateFlagUsecase(app.Flags, app.SnapshotPointer),
		usecase.NewDeleteFlagUsecase(app.Flags, app.SnapshotPointer),
		app.Idempotency,
	)
	evalHandler := handler.NewEvaluateHandler(
		usecase.NewEvaluateFlagsUsecase(app.ScopedFlags),
	)

	dashboardHandler := handler.NewDashboardHandler(
		app.AdminRead,
		usecase.NewRollbackFlagUsecase(app.Flags, app.AdminRead, app.SnapshotPointer),
	)

	public := v1.Group("/public")
	public.Use(middleware.RateLimit(20, 60))
	public.Use(middleware.StaticAPIKey("X-Public-Api-Key", app.Config.PublicReadAPIKey))
	publicHandler := handler.NewPublicFlagsHandler(listFlags)
	public.GET("/projects/:projectId/environments/:environmentKey/flags", publicHandler.List)

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
	v1.GET("/projects/:projectId/environments/:environmentKey/flags/evaluate", evalHandler.EvaluateGET)

	v1.GET("/projects/:projectId/environments/:environmentKey/flags/:flagId/versions", dashboardHandler.ListFlagVersions)
	v1.GET("/projects/:projectId/environments/:environmentKey/flags/:flagId/versions/diff", dashboardHandler.DiffFlagVersions)
	v1.POST("/projects/:projectId/environments/:environmentKey/flags/:flagId/rollback", dashboardHandler.Rollback)

	v1.GET("/projects/:projectId/audit", dashboardHandler.ListAudit)

	wsHandler := handler.NewBroadcastWSHandler(app.Config, app.BroadcastHub, app.ScopedFlags)
	v1.GET("/broadcast/flags", wsHandler.Connect)
}
