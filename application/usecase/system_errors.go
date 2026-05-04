package usecase

import apperror "flag-gen/pkg/error"

var (
	errInstallNotAllowed  = apperror.New(apperror.StatusConflict, "install_not_allowed", nil)
	errInstallLockTimeout = apperror.New(apperror.StatusConflict, "install_lock_timeout", nil)
	errInstallRace        = apperror.New(apperror.StatusConflict, "install_state_changed", nil)
	errInstallBadAdmin    = apperror.New(apperror.StatusUnprocessableEntity, "invalid_admin", nil)
	errMigrateForbidden   = apperror.New(apperror.StatusForbidden, "migrate_forbidden", nil)
	errMigrateNotNeeded   = apperror.New(apperror.StatusConflict, "migrate_not_needed", nil)
	errMigrateLockTimeout = apperror.New(apperror.StatusConflict, "migrate_lock_timeout", nil)
)
