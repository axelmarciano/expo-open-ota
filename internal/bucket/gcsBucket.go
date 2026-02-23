package bucket

import (
    "bytes"
    "context"
    "errors"
    "expo-open-ota/internal/services"
    "expo-open-ota/internal/types"
    "fmt"
    "io"
    "runtime"
    "sort"
    "strconv"
    "strings"
    "sync"
    "time"

    "cloud.google.com/go/storage"
    "google.golang.org/api/iterator"
)

type GCSBucket struct {
    BucketName string
}

func (b *GCSBucket) bucketHandle(ctx context.Context) (*storage.BucketHandle, error) {
    if b.BucketName == "" {
        return nil, errors.New("BucketName not set")
    }
    client, err := services.GetGCSClient()
    if err != nil {
        return nil, err
    }
    return client.Bucket(b.BucketName), nil
}

func (b *GCSBucket) DeleteUpdateFolder(branch, runtimeVersion, updateId string) error {
    ctx := context.Background()
    bh, err := b.bucketHandle(ctx)
    if err != nil {
        return err
    }
    prefix := fmt.Sprintf("%s/%s/%s/", branch, runtimeVersion, updateId)
    it := bh.Objects(ctx, &storage.Query{Prefix: prefix})
    sem := make(chan struct{}, runtime.NumCPU())
    var wg sync.WaitGroup
    errCh := make(chan error, 16)
    for {
        attrs, err := it.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            return fmt.Errorf("failed to list objects: %w", err)
        }
        if attrs.Name == "" { // prefix entry
            continue
        }
        wg.Add(1)
        sem <- struct{}{}
        go func(name string) {
            defer wg.Done()
            defer func() { <-sem }()
            if err := bh.Object(name).Delete(ctx); err != nil {
                errCh <- fmt.Errorf("failed to delete object %s: %w", name, err)
            }
        }(attrs.Name)
    }
    wg.Wait()
    close(errCh)
    for e := range errCh {
        if e != nil {
            return e
        }
    }
    return nil
}

func (b *GCSBucket) GetRuntimeVersions(branch string) ([]RuntimeVersionWithStats, error) {
    ctx := context.Background()
    bh, err := b.bucketHandle(ctx)
    if err != nil {
        return nil, err
    }
    q := &storage.Query{Prefix: branch + "/", Delimiter: "/"}
    it := bh.Objects(ctx, q)
    var runtimeVersions []RuntimeVersionWithStats
    for {
        attrs, err := it.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            return nil, fmt.Errorf("list runtime versions: %w", err)
        }
        if attrs.Prefix == "" { // skip objects
            continue
        }
        rvPrefix := attrs.Prefix // e.g., branch/runtime/
        rv := strings.TrimSuffix(strings.TrimPrefix(rvPrefix, branch+"/"), "/")

        // list update folders under this runtimeVersion
        updatesIt := bh.Objects(ctx, &storage.Query{Prefix: rvPrefix, Delimiter: "/"})
        var updateTimestamps []int64
        for {
            uattrs, err := updatesIt.Next()
            if err == iterator.Done {
                break
            }
            if err != nil {
                return nil, fmt.Errorf("list updates: %w", err)
            }
            if uattrs.Prefix == "" {
                continue
            }
            upd := strings.TrimSuffix(strings.TrimPrefix(uattrs.Prefix, rvPrefix), "/")
            ts, err := strconv.ParseInt(upd, 10, 64)
            if err == nil {
                updateTimestamps = append(updateTimestamps, ts)
            }
        }
        if len(updateTimestamps) == 0 {
            continue
        }
        sort.Slice(updateTimestamps, func(i, j int) bool { return updateTimestamps[i] < updateTimestamps[j] })
        runtimeVersions = append(runtimeVersions, RuntimeVersionWithStats{
            RuntimeVersion:  rv,
            CreatedAt:       time.UnixMilli(updateTimestamps[0]).UTC().Format(time.RFC3339),
            LastUpdatedAt:   time.UnixMilli(updateTimestamps[len(updateTimestamps)-1]).UTC().Format(time.RFC3339),
            NumberOfUpdates: len(updateTimestamps),
        })
    }
    return runtimeVersions, nil
}

func (b *GCSBucket) GetBranches() ([]string, error) {
    ctx := context.Background()
    bh, err := b.bucketHandle(ctx)
    if err != nil {
        return nil, err
    }
    it := bh.Objects(ctx, &storage.Query{Delimiter: "/"})
    var branches []string
    for {
        attrs, err := it.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            return nil, fmt.Errorf("list branches: %w", err)
        }
        if attrs.Prefix == "" {
            continue
        }
        prefix := strings.TrimSuffix(attrs.Prefix, "/")
        branches = append(branches, prefix)
    }
    return branches, nil
}

func (b *GCSBucket) GetUpdates(branch string, runtimeVersion string) ([]types.Update, error) {
    ctx := context.Background()
    bh, err := b.bucketHandle(ctx)
    if err != nil {
        return nil, err
    }
    prefix := branch + "/" + runtimeVersion + "/"
    it := bh.Objects(ctx, &storage.Query{Prefix: prefix, Delimiter: "/"})
    var updates []types.Update
    for {
        attrs, err := it.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            return nil, fmt.Errorf("list updates: %w", err)
        }
        if attrs.Prefix == "" {
            continue
        }
        name := strings.TrimSuffix(strings.TrimPrefix(attrs.Prefix, prefix), "/")
        if id, err := strconv.ParseInt(name, 10, 64); err == nil {
            updates = append(updates, types.Update{
                Branch:         branch,
                RuntimeVersion: runtimeVersion,
                UpdateId:       strconv.FormatInt(id, 10),
                CreatedAt:      time.Duration(id) * time.Millisecond,
            })
        }
    }
    return updates, nil
}

func (b *GCSBucket) GetFile(update types.Update, assetPath string) (*types.BucketFile, error) {
    ctx := context.Background()
    bh, err := b.bucketHandle(ctx)
    if err != nil {
        return nil, err
    }
    key := update.Branch + "/" + update.RuntimeVersion + "/" + update.UpdateId + "/" + assetPath
    obj := bh.Object(key)
    r, err := obj.NewReader(ctx)
    if err != nil {
        if errors.Is(err, storage.ErrObjectNotExist) {
            return nil, nil
        }
        return nil, fmt.Errorf("GetObject error: %w", err)
    }
    attrs, _ := obj.Attrs(ctx)
    var created time.Time
    if attrs != nil {
        created = attrs.Updated
    }
    return &types.BucketFile{Reader: r, CreatedAt: created}, nil
}

func (b *GCSBucket) RequestUploadUrlForFileUpdate(branch string, runtimeVersion string, updateId string, fileName string) (string, error) {
    if b.BucketName == "" {
        return "", errors.New("BucketName not set")
    }
    key := fmt.Sprintf("%s/%s/%s/%s", branch, runtimeVersion, updateId, fileName)
    url, err := services.GCSSignedURL(b.BucketName, key, "PUT", "", 15*time.Minute)
    if err != nil {
        return "", fmt.Errorf("error generating signed URL: %w", err)
    }
    return url, nil
}

func (b *GCSBucket) UploadFileIntoUpdate(update types.Update, fileName string, file io.Reader) error {
    ctx := context.Background()
    bh, err := b.bucketHandle(ctx)
    if err != nil {
        return err
    }
    key := fmt.Sprintf("%s/%s/%s/%s", update.Branch, update.RuntimeVersion, update.UpdateId, fileName)
    w := bh.Object(key).NewWriter(ctx)
    if _, err := io.Copy(w, file); err != nil {
        _ = w.Close()
        return err
    }
    if err := w.Close(); err != nil {
        return err
    }
    return nil
}

func (b *GCSBucket) CreateUpdateFrom(previousUpdate *types.Update, newUpdateId string) (*types.Update, error) {
    if b.BucketName == "" {
        return nil, errors.New("BucketName not set")
    }
    if previousUpdate == nil {
        return nil, errors.New("previousUpdate is nil")
    }
    if previousUpdate.UpdateId == "" {
        return nil, errors.New("previousUpdate.UpdateId is empty")
    }
    if newUpdateId == "" {
        return nil, errors.New("newUpdateId is empty")
    }
    ctx := context.Background()
    bh, err := b.bucketHandle(ctx)
    if err != nil {
        return nil, err
    }
    sourcePrefix := fmt.Sprintf("%s/%s/%s/", previousUpdate.Branch, previousUpdate.RuntimeVersion, previousUpdate.UpdateId)
    targetPrefix := fmt.Sprintf("%s/%s/%s/", previousUpdate.Branch, previousUpdate.RuntimeVersion, newUpdateId)

    it := bh.Objects(ctx, &storage.Query{Prefix: sourcePrefix})
    var wg sync.WaitGroup
    errChan := make(chan error, 16)
    sem := make(chan struct{}, runtime.NumCPU())

    for {
        attrs, err := it.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            return nil, fmt.Errorf("failed to list objects: %w", err)
        }
        if attrs.Name == "" {
            continue
        }
        relPath := strings.TrimPrefix(attrs.Name, sourcePrefix)
        if relPath == "update-metadata.json" || relPath == ".check" || strings.HasSuffix(relPath, "/") {
            continue
        }
        src := attrs.Name
        dst := targetPrefix + relPath

        wg.Add(1)
        go func(srcKey, dstKey string) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()
            cop := bh.Object(dstKey).CopierFrom(bh.Object(srcKey))
            if _, err := cop.Run(ctx); err != nil {
                errChan <- fmt.Errorf("copy %s -> %s: %w", srcKey, dstKey, err)
            }
        }(src, dst)
    }

    wg.Wait()
    close(errChan)
    for e := range errChan {
        if e != nil {
            return nil, e
        }
    }

    updateId, err := strconv.ParseInt(newUpdateId, 10, 64)
    if err != nil {
        return nil, fmt.Errorf("error parsing update ID: %w", err)
    }
    return &types.Update{
        Branch:         previousUpdate.Branch,
        RuntimeVersion: previousUpdate.RuntimeVersion,
        UpdateId:       newUpdateId,
        CreatedAt:      time.Duration(updateId) * time.Millisecond,
    }, nil
}

func (b *GCSBucket) RetrieveMigrationHistory() ([]string, error) {
    ctx := context.Background()
    bh, err := b.bucketHandle(ctx)
    if err != nil {
        return nil, err
    }
    obj := bh.Object(".migrationhistory")
    r, err := obj.NewReader(ctx)
    if err != nil {
        if errors.Is(err, storage.ErrObjectNotExist) {
            return nil, nil
        }
        return nil, err
    }
    defer r.Close()
    var migrations []string
    buf := new(bytes.Buffer)
    if _, err := io.Copy(buf, r); err != nil {
        return nil, err
    }
    for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
        if line != "" {
            migrations = append(migrations, line)
        }
    }
    return migrations, nil
}

func (b *GCSBucket) ApplyMigration(migrationId string) error {
    ctx := context.Background()
    bh, err := b.bucketHandle(ctx)
    if err != nil {
        return err
    }
    history, err := b.RetrieveMigrationHistory()
    if err != nil {
        return fmt.Errorf("RetrieveMigrationHistory error: %w", err)
    }
    for _, id := range history {
        if id == migrationId {
            return nil
        }
    }
    current := strings.Join(history, "\n")
    if current != "" {
        current += "\n"
    }
    data := []byte(current + migrationId + "\n")
    w := bh.Object(".migrationhistory").NewWriter(ctx)
    if _, err := w.Write(data); err != nil {
        _ = w.Close()
        return err
    }
    return w.Close()
}

func (b *GCSBucket) RemoveMigrationFromHistory(migrationId string) error {
    ctx := context.Background()
    bh, err := b.bucketHandle(ctx)
    if err != nil {
        return err
    }
    history, err := b.RetrieveMigrationHistory()
    if err != nil {
        return fmt.Errorf("RetrieveMigrationHistory error: %w", err)
    }
    // If not present, nothing to do
    found := false
    var filtered []string
    for _, id := range history {
        if id == migrationId {
            found = true
            continue
        }
        filtered = append(filtered, id)
    }
    if !found {
        return nil
    }
    content := ""
    if len(filtered) > 0 {
        content = strings.Join(filtered, "\n") + "\n"
    }
    w := bh.Object(".migrationhistory").NewWriter(ctx)
    if _, err := w.Write([]byte(content)); err != nil {
        _ = w.Close()
        return err
    }
    return w.Close()
}
