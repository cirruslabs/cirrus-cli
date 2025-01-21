// Code generated by ogen, DO NOT EDIT.

package api

import (
	"context"
)

// Handler handles operations described by OpenAPI v3 specification.
type Handler interface {
	// Authenticate implements authenticate operation.
	//
	// This endpoint returns API tokens for a given email and password.
	//
	// POST /api/auth
	Authenticate(ctx context.Context, req OptAuthenticateReq) (AuthenticateRes, error)
	// CacheArtifactExists implements cacheArtifactExists operation.
	//
	// This endpoint checks if an artifact exists in the cache. It returns a 404 status code if the
	// artifact does not exist.
	//
	// Deprecated: schema marks this operation as deprecated.
	//
	// GET /api/cache/exists
	CacheArtifactExists(ctx context.Context, params CacheArtifactExistsParams) (CacheArtifactExistsRes, error)
	// CancelInvitation implements cancelInvitation operation.
	//
	// Cancels an invitation for a given invitee email and an organization.
	//
	// DELETE /api/organizations/{organization_name}/invitations
	CancelInvitation(ctx context.Context, req OptCancelInvitationReq, params CancelInvitationParams) (CancelInvitationRes, error)
	// CleanCache implements cleanCache operation.
	//
	// Cleans cache for a given project.
	//
	// PUT /api/projects/{account_handle}/{project_handle}/cache/clean
	CleanCache(ctx context.Context, params CleanCacheParams) (CleanCacheRes, error)
	// CompleteAnalyticsArtifactMultipartUpload implements completeAnalyticsArtifactMultipartUpload operation.
	//
	// Given the upload ID and all the parts with their ETags, this endpoint completes the multipart
	// upload.
	//
	// POST /api/runs/{run_id}/complete
	CompleteAnalyticsArtifactMultipartUpload(ctx context.Context, req OptCompleteAnalyticsArtifactMultipartUploadReq, params CompleteAnalyticsArtifactMultipartUploadParams) (CompleteAnalyticsArtifactMultipartUploadRes, error)
	// CompleteAnalyticsArtifactsUploads implements completeAnalyticsArtifactsUploads operation.
	//
	// Given a command event, it marks all artifact uploads as finished and does extra processing of a
	// given command run, such as test flakiness detection.
	//
	// PUT /api/runs/{run_id}/complete_artifacts_uploads
	CompleteAnalyticsArtifactsUploads(ctx context.Context, req OptCompleteAnalyticsArtifactsUploadsReq, params CompleteAnalyticsArtifactsUploadsParams) (CompleteAnalyticsArtifactsUploadsRes, error)
	// CompleteCacheArtifactMultipartUpload implements completeCacheArtifactMultipartUpload operation.
	//
	// Given the upload ID and all the parts with their ETags, this endpoint completes the multipart
	// upload. The cache will then be able to serve the artifact.
	//
	// POST /api/cache/multipart/complete
	CompleteCacheArtifactMultipartUpload(ctx context.Context, req OptCompleteCacheArtifactMultipartUploadReq, params CompleteCacheArtifactMultipartUploadParams) (CompleteCacheArtifactMultipartUploadRes, error)
	// CompletePreviewsMultipartUpload implements completePreviewsMultipartUpload operation.
	//
	// Given the upload ID and all the parts with their ETags, this endpoint completes the multipart
	// upload.
	//
	// POST /api/projects/{account_handle}/{project_handle}/previews/complete
	CompletePreviewsMultipartUpload(ctx context.Context, req OptCompletePreviewsMultipartUploadReq, params CompletePreviewsMultipartUploadParams) (CompletePreviewsMultipartUploadRes, error)
	// CreateAccountToken implements createAccountToken operation.
	//
	// This endpoint returns a new account token.
	//
	// POST /api/accounts/{account_handle}/tokens
	CreateAccountToken(ctx context.Context, req OptCreateAccountTokenReq, params CreateAccountTokenParams) (CreateAccountTokenRes, error)
	// CreateCommandEvent implements createCommandEvent operation.
	//
	// Create a a new command analytics event.
	//
	// POST /api/analytics
	CreateCommandEvent(ctx context.Context, req OptCreateCommandEventReq, params CreateCommandEventParams) (CreateCommandEventRes, error)
	// CreateInvitation implements createInvitation operation.
	//
	// Invites a user with a given email to a given organization.
	//
	// POST /api/organizations/{organization_name}/invitations
	CreateInvitation(ctx context.Context, req OptCreateInvitationReq, params CreateInvitationParams) (CreateInvitationRes, error)
	// CreateOrganization implements createOrganization operation.
	//
	// Creates an organization with the given name.
	//
	// POST /api/organizations
	CreateOrganization(ctx context.Context, req OptCreateOrganizationReq) (CreateOrganizationRes, error)
	// CreateProject implements createProject operation.
	//
	// Create a new project.
	//
	// POST /api/projects
	CreateProject(ctx context.Context, req OptCreateProjectReq) (CreateProjectRes, error)
	// CreateProjectToken implements createProjectToken operation.
	//
	// This endpoint returns a new project token.
	//
	// POST /api/projects/{account_handle}/{project_handle}/tokens
	CreateProjectToken(ctx context.Context, params CreateProjectTokenParams) (CreateProjectTokenRes, error)
	// DeleteOrganization implements deleteOrganization operation.
	//
	// Deletes the organization with the given name.
	//
	// DELETE /api/organizations/{organization_name}
	DeleteOrganization(ctx context.Context, params DeleteOrganizationParams) (DeleteOrganizationRes, error)
	// DeleteProject implements deleteProject operation.
	//
	// Deletes a project with a given id.
	//
	// DELETE /api/projects/{id}
	DeleteProject(ctx context.Context, params DeleteProjectParams) (DeleteProjectRes, error)
	// DownloadCacheArtifact implements downloadCacheArtifact operation.
	//
	// This endpoint returns a signed URL that can be used to download an artifact from the cache.
	//
	// GET /api/cache
	DownloadCacheArtifact(ctx context.Context, params DownloadCacheArtifactParams) (DownloadCacheArtifactRes, error)
	// DownloadPreview implements downloadPreview operation.
	//
	// This endpoint returns a preview with a given id, including the url to download the preview.
	//
	// GET /api/projects/{account_handle}/{project_handle}/previews/{preview_id}
	DownloadPreview(ctx context.Context, params DownloadPreviewParams) (DownloadPreviewRes, error)
	// GenerateAnalyticsArtifactMultipartUploadURL implements generateAnalyticsArtifactMultipartUploadURL operation.
	//
	// Given an upload ID and a part number, this endpoint returns a signed URL that can be used to
	// upload a part of a multipart upload. The URL is short-lived and expires in 120 seconds.
	//
	// POST /api/runs/{run_id}/generate-url
	GenerateAnalyticsArtifactMultipartUploadURL(ctx context.Context, req OptGenerateAnalyticsArtifactMultipartUploadURLReq, params GenerateAnalyticsArtifactMultipartUploadURLParams) (GenerateAnalyticsArtifactMultipartUploadURLRes, error)
	// GenerateCacheArtifactMultipartUploadURL implements generateCacheArtifactMultipartUploadURL operation.
	//
	// Given an upload ID and a part number, this endpoint returns a signed URL that can be used to
	// upload a part of a multipart upload. The URL is short-lived and expires in 120 seconds.
	//
	// POST /api/cache/multipart/generate-url
	GenerateCacheArtifactMultipartUploadURL(ctx context.Context, params GenerateCacheArtifactMultipartUploadURLParams) (GenerateCacheArtifactMultipartUploadURLRes, error)
	// GeneratePreviewsMultipartUploadURL implements generatePreviewsMultipartUploadURL operation.
	//
	// Given an upload ID and a part number, this endpoint returns a signed URL that can be used to
	// upload a part of a multipart upload. The URL is short-lived and expires in 120 seconds.
	//
	// POST /api/projects/{account_handle}/{project_handle}/previews/generate-url
	GeneratePreviewsMultipartUploadURL(ctx context.Context, req OptGeneratePreviewsMultipartUploadURLReq, params GeneratePreviewsMultipartUploadURLParams) (GeneratePreviewsMultipartUploadURLRes, error)
	// GetCacheActionItem implements getCacheActionItem operation.
	//
	// This endpoint gets an item from the action cache.
	//
	// GET /api/projects/{account_handle}/{project_handle}/cache/ac/{hash}
	GetCacheActionItem(ctx context.Context, params GetCacheActionItemParams) (GetCacheActionItemRes, error)
	// GetDeviceCode implements getDeviceCode operation.
	//
	// This endpoint returns a token for a given device code if the device code is authenticated.
	//
	// GET /api/auth/device_code/{device_code}
	GetDeviceCode(ctx context.Context, params GetDeviceCodeParams) (GetDeviceCodeRes, error)
	// ListOrganizations implements listOrganizations operation.
	//
	// Returns all the organizations the authenticated subject is part of.
	//
	// GET /api/organizations
	ListOrganizations(ctx context.Context) (ListOrganizationsRes, error)
	// ListPreviews implements listPreviews operation.
	//
	// This endpoint returns a list of previews for a given project.
	//
	// GET /api/projects/{account_handle}/{project_handle}/previews
	ListPreviews(ctx context.Context, params ListPreviewsParams) (ListPreviewsRes, error)
	// ListProjectTokens implements listProjectTokens operation.
	//
	// This endpoint returns all tokens for a given project.
	//
	// GET /api/projects/{account_handle}/{project_handle}/tokens
	ListProjectTokens(ctx context.Context, params ListProjectTokensParams) (ListProjectTokensRes, error)
	// ListProjects implements listProjects operation.
	//
	// List projects the authenticated user has access to.
	//
	// GET /api/projects
	ListProjects(ctx context.Context) (ListProjectsRes, error)
	// ListRuns implements listRuns operation.
	//
	// List runs associated with a given project.
	//
	// GET /api/projects/{account_handle}/{project_handle}/runs
	ListRuns(ctx context.Context, params ListRunsParams) (ListRunsRes, error)
	// RefreshToken implements refreshToken operation.
	//
	// This endpoint returns new tokens for a given refresh token if the refresh token is valid.
	//
	// POST /api/auth/refresh_token
	RefreshToken(ctx context.Context, req OptRefreshTokenReq) (RefreshTokenRes, error)
	// RevokeProjectToken implements revokeProjectToken operation.
	//
	// Revokes a project token.
	//
	// DELETE /api/projects/{account_handle}/{project_handle}/tokens/{id}
	RevokeProjectToken(ctx context.Context, params RevokeProjectTokenParams) (RevokeProjectTokenRes, error)
	// ShowOrganization implements showOrganization operation.
	//
	// Returns the organization with the given identifier.
	//
	// GET /api/organizations/{organization_name}
	ShowOrganization(ctx context.Context, params ShowOrganizationParams) (ShowOrganizationRes, error)
	// ShowOrganizationUsage implements showOrganizationUsage operation.
	//
	// Returns the usage of the organization with the given identifier. (e.g. number of remote cache hits).
	//
	// GET /api/organizations/{organization_name}/usage
	ShowOrganizationUsage(ctx context.Context, params ShowOrganizationUsageParams) (ShowOrganizationUsageRes, error)
	// ShowProject implements showProject operation.
	//
	// Returns a project based on the handle.
	//
	// GET /api/projects/{account_handle}/{project_handle}
	ShowProject(ctx context.Context, params ShowProjectParams) (ShowProjectRes, error)
	// StartAnalyticsArtifactMultipartUpload implements startAnalyticsArtifactMultipartUpload operation.
	//
	// The endpoint returns an upload ID that can be used to generate URLs for the individual parts and
	// complete the upload.
	//
	// POST /api/runs/{run_id}/start
	StartAnalyticsArtifactMultipartUpload(ctx context.Context, req OptCommandEventArtifact, params StartAnalyticsArtifactMultipartUploadParams) (StartAnalyticsArtifactMultipartUploadRes, error)
	// StartCacheArtifactMultipartUpload implements startCacheArtifactMultipartUpload operation.
	//
	// The endpoint returns an upload ID that can be used to generate URLs for the individual parts and
	// complete the upload.
	//
	// POST /api/cache/multipart/start
	StartCacheArtifactMultipartUpload(ctx context.Context, params StartCacheArtifactMultipartUploadParams) (StartCacheArtifactMultipartUploadRes, error)
	// StartPreviewsMultipartUpload implements startPreviewsMultipartUpload operation.
	//
	// The endpoint returns an upload ID that can be used to generate URLs for the individual parts and
	// complete the upload.
	//
	// POST /api/projects/{account_handle}/{project_handle}/previews/start
	StartPreviewsMultipartUpload(ctx context.Context, req OptStartPreviewsMultipartUploadReq, params StartPreviewsMultipartUploadParams) (StartPreviewsMultipartUploadRes, error)
	// UpdateAccount implements updateAccount operation.
	//
	// Updates the given account.
	//
	// PATCH /api/accounts/{account_handle}
	UpdateAccount(ctx context.Context, req OptUpdateAccountReq, params UpdateAccountParams) (UpdateAccountRes, error)
	// UpdateOrganization implements updateOrganization operation.
	//
	// Updates an organization with given parameters.
	//
	// PUT /api/organizations/{organization_name}
	UpdateOrganization(ctx context.Context, req OptUpdateOrganizationReq, params UpdateOrganizationParams) (UpdateOrganizationRes, error)
	// UpdateOrganization2 implements updateOrganization (2) operation.
	//
	// Updates an organization with given parameters.
	//
	// PATCH /api/organizations/{organization_name}
	UpdateOrganization2(ctx context.Context, req OptUpdateOrganization2Req, params UpdateOrganization2Params) (UpdateOrganization2Res, error)
	// UpdateOrganizationMember implements updateOrganizationMember operation.
	//
	// Updates a member in a given organization.
	//
	// PUT /api/organizations/{organization_name}/members/{user_name}
	UpdateOrganizationMember(ctx context.Context, req OptUpdateOrganizationMemberReq, params UpdateOrganizationMemberParams) (UpdateOrganizationMemberRes, error)
	// UpdateProject implements updateProject operation.
	//
	// Updates a project with given parameters.
	//
	// PUT /api/projects/{account_handle}/{project_handle}
	UpdateProject(ctx context.Context, req OptUpdateProjectReq, params UpdateProjectParams) (UpdateProjectRes, error)
	// UploadCacheActionItem implements uploadCacheActionItem operation.
	//
	// The endpoint caches a given action item without uploading a file. To upload files, use the
	// multipart upload instead.
	//
	// POST /api/projects/{account_handle}/{project_handle}/cache/ac
	UploadCacheActionItem(ctx context.Context, req OptUploadCacheActionItemReq, params UploadCacheActionItemParams) (UploadCacheActionItemRes, error)
	// UploadPreviewIcon implements uploadPreviewIcon operation.
	//
	// The endpoint uploads a preview icon.
	//
	// POST /api/projects/{account_handle}/{project_handle}/previews/{preview_id}/icons
	UploadPreviewIcon(ctx context.Context, params UploadPreviewIconParams) (UploadPreviewIconRes, error)
}

// Server implements http server based on OpenAPI v3 specification and
// calls Handler to handle requests.
type Server struct {
	h   Handler
	sec SecurityHandler
	baseServer
}

// NewServer creates new Server.
func NewServer(h Handler, sec SecurityHandler, opts ...ServerOption) (*Server, error) {
	s, err := newServerConfig(opts...).baseServer()
	if err != nil {
		return nil, err
	}
	return &Server{
		h:          h,
		sec:        sec,
		baseServer: s,
	}, nil
}
