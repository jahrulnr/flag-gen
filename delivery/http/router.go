package http

import (
	"flag-gen/application/usecase"
	"flag-gen/delivery/http/handler"
	"flag-gen/delivery/http/middleware"
	"flag-gen/infrastructure/di"

	"github.com/gin-gonic/gin"
)

func RegisterV1Routes(r *gin.Engine, app *di.ApplicationState) {
	RegisterRoutes(r.Group("/v1"), app, nil)
}

func RegisterAPIRoutes(r *gin.Engine, app *di.ApplicationState, csrf *middleware.CSRFProtection) {
	api := r.Group("/api")
	api.Use(csrf.Middleware())
	RegisterRoutes(api, app, csrf)
}

func RegisterRoutes(group *gin.RouterGroup, app *di.ApplicationState, csrf *middleware.CSRFProtection) {
	systemHandler := handler.NewSystemHandler(app.System)
	group.GET("/system/status", systemHandler.Status)
	group.GET("/system/metrics", systemHandler.Metrics)
	group.POST("/system/install", systemHandler.Install)
	group.POST("/system/migrate", systemHandler.Migrate)

	if csrf != nil {
		securityHandler := handler.NewSecurityHandler(csrf)
		group.GET("/security/csrf", securityHandler.CSRF)
	}

	listFlags := usecase.NewListFlagsUsecase(app.Flags)

	projectsHandler := handler.NewProjectsHandler(
		usecase.NewListProjectsUsecase(app.Projects),
		usecase.NewCreateProjectUsecase(app.Projects),
		usecase.NewDeleteProjectUsecase(app.Projects, app.Environments, app.Segments),
	)
	envHandler := handler.NewEnvironmentsHandler(
		usecase.NewListEnvironmentsUsecase(app.Environments),
		usecase.NewCreateEnvironmentUsecase(app.Environments),
		usecase.NewDeleteEnvironmentUsecase(app.Environments, app.Flags),
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

	public := group.Group("/public")
	public.Use(middleware.RateLimit(20, 60))
	public.Use(middleware.StaticAPIKey("X-Public-Api-Key", app.Config.PublicReadAPIKey))
	publicHandler := handler.NewPublicFlagsHandler(listFlags)
	public.GET("/projects/:projectId/environments/:environmentKey/flags", publicHandler.List)

	group.GET("/projects", projectsHandler.List)
	group.POST("/projects", projectsHandler.Create)
	group.DELETE("/projects/:projectId", projectsHandler.Delete)

	group.GET("/projects/:projectId/environments", envHandler.List)
	group.POST("/projects/:projectId/environments", envHandler.Create)
	group.DELETE("/projects/:projectId/environments/:environmentKey", envHandler.Delete)

	group.GET("/projects/:projectId/segments", segmentsHandler.List)
	group.POST("/projects/:projectId/segments", segmentsHandler.Create)
	group.PATCH("/projects/:projectId/segments/:segmentId", segmentsHandler.Update)
	group.DELETE("/projects/:projectId/segments/:segmentId", segmentsHandler.Delete)

	group.GET("/projects/:projectId/environments/:environmentKey/flags", flagsHandler.List)
	group.POST("/projects/:projectId/environments/:environmentKey/flags", flagsHandler.Create)
	// Static /flags/evaluate before /flags/:flagKey so "evaluate" is never captured as a key.
	group.POST("/projects/:projectId/environments/:environmentKey/flags/evaluate", evalHandler.Evaluate)
	group.GET("/projects/:projectId/environments/:environmentKey/flags/evaluate", evalHandler.EvaluateGET)

	group.GET("/projects/:projectId/environments/:environmentKey/flags/:flagKey", flagsHandler.GetByKey)
	group.PATCH("/projects/:projectId/environments/:environmentKey/flags/:flagKey", flagsHandler.Update)
	group.DELETE("/projects/:projectId/environments/:environmentKey/flags/:flagKey", flagsHandler.Delete)

	group.GET("/projects/:projectId/environments/:environmentKey/flags/:flagKey/versions", dashboardHandler.ListFlagVersions)
	group.GET("/projects/:projectId/environments/:environmentKey/flags/:flagKey/versions/diff", dashboardHandler.DiffFlagVersions)
	group.POST("/projects/:projectId/environments/:environmentKey/flags/:flagKey/rollback", dashboardHandler.Rollback)

	group.GET("/projects/:projectId/audit", dashboardHandler.ListAudit)

	wsHandler := handler.NewBroadcastWSHandler(app.Config, app.BroadcastHub, app.ScopedFlags)
	group.GET("/broadcast/flags", wsHandler.Connect)
}
