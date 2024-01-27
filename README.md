# Thread-safe 'slog' wrapper for local development

### Import package
```bash
    go get github.com/humanbelnik/logit
```
### Configure logger
```go
    logger := slog.New(logit.NewHandler(slog.LevelDebug))
```
