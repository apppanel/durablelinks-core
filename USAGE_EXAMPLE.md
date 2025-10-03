```go
// In your API package
func (h *handler) CreateLink(w http.ResponseWriter, r *http.Request) {
    // Parse and validate in one step - handled by core!
    createReq, err := models.ParseAndValidateCreateRequest(r.Body)
    if err != nil {
        // Check if it's validation errors
        if validationErrs, ok := err.(models.ValidationErrors); ok {
            WriteValidationErrorResponse(w, http.StatusBadRequest, "INVALID_ARGUMENT", validationErrs.Errors)
            return
        }
        // Other errors
        WriteErrorResponse(w, http.StatusInternalServerError, err.Error(), "INTERNAL")
        return
    }

    // Create the link
    shortLinkResp, err := h.linkService.CreateDurableLink(r.Context(), *createReq, nil, h.tenantCfg)
    switch {
    case errors.Is(err, service.ErrDomainLinkNotAllowed):
        WriteErrorResponse(w, http.StatusBadRequest, "'link' parameter contains a host that is not in the allow list", "INVALID_ARGUMENT")
    case err != nil:
        log.Error().Err(err).Msg("Failed to create dynamic link")
        WriteErrorResponse(w, http.StatusInternalServerError, "Failed to create link", "INTERNAL")
    default:
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(shortLinkResp)
    }
}
```

## Benefits

1. **API layer is just transport** - No validation logic, just error handling
2. **Core handles everything** - JSON parsing + validation in one call
3. **Reusable across transports** - Use same validation in HTTP, gRPC, CLI, etc.
4. **Type-safe errors** - `ValidationErrors` type is easy to check and handle
5. **Consistent error formatting** - All validation errors formatted the same way

## For other transport layers (gRPC, CLI, etc.)

```go
// gRPC example
func (s *grpcServer) CreateLink(ctx context.Context, req *pb.CreateLinkRequest) (*pb.CreateLinkResponse, error) {
    // Convert protobuf to JSON
    jsonBytes, _ := json.Marshal(req)

    // Parse and validate using the same core function!
    createReq, err := models.ParseAndValidateCreateRequest(bytes.NewReader(jsonBytes))
    if err != nil {
        if validationErrs, ok := err.(models.ValidationErrors); ok {
            // Return gRPC InvalidArgument with validation details
            return nil, status.Errorf(codes.InvalidArgument, validationErrs.Error())
        }
        return nil, status.Error(codes.Internal, err.Error())
    }

    // Use the validated request
    result, err := s.linkService.CreateDurableLink(ctx, *createReq, nil, s.tenantCfg)
    // ...
}
```

## Core Provides

- `models.ParseAndValidateCreateRequest(io.Reader)` - Parse JSON and validate
- `models.ValidationErrors` - Structured validation errors
- `models.ValidationError` - Single field error with Field, Tag, Message
