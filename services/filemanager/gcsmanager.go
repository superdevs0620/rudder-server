package filemanager

import (
	"context"
	"fmt"
	"google.golang.org/api/iterator"
	"io"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func objectURL(objAttrs *storage.ObjectAttrs) string {
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", objAttrs.Bucket, objAttrs.Name)
}

func (manager *GCSManager) Upload(file *os.File, prefixes ...string) (UploadOutput, error) {
	ctx := context.Background()
	var client *storage.Client
	var err error
	if manager.Config.Credentials == "" {
		client, err = storage.NewClient(ctx)
	} else {
		client, err = storage.NewClient(ctx, option.WithCredentialsJSON([]byte(manager.Config.Credentials)))
	}

	if err != nil {
		return UploadOutput{}, err
	}
	splitFileName := strings.Split(file.Name(), "/")
	fileName := ""
	if len(prefixes) > 0 {
		fileName = strings.Join(prefixes[:], "/") + "/"
	}
	fileName += splitFileName[len(splitFileName)-1]
	if manager.Config.Prefix != "" {
		if manager.Config.Prefix[len(manager.Config.Prefix)-1:] == "/" {
			fileName = manager.Config.Prefix + fileName
		} else {
			fileName = manager.Config.Prefix + "/" + fileName
		}
	}
	bh := client.Bucket(manager.Config.Bucket)
	obj := bh.Object(fileName)
	w := obj.NewWriter(ctx)
	if _, err := io.Copy(w, file); err != nil {
		return UploadOutput{}, err
	}
	if err := w.Close(); err != nil {
		return UploadOutput{}, err
	}

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return UploadOutput{}, err
	}
	return UploadOutput{Location: objectURL(attrs), ObjectName: fileName}, err
}

func (manager *GCSManager) GetStorageDateFormat(prefixes ...string) (dateFormat string, err error) {
	dateFormat = "YYYY-MM-DD"
	prefix := strings.Join(prefixes[0:2],"/")
	gcsObjects, err := manager.ListFilesWithPrefix(prefix,1)
	if err != nil {
		return
	}
	if len(gcsObjects) == 0 {
		return
	}
	date := strings.Split(gcsObjects[0], "/")[2]
	allDateLayouts := map[string]string{
		"01-02-2006" : "MM-DD-YYYY",
		"2006-01-02" : "YYYY-MM-DD",
		//"02-01-2006" : "DD-MM-YYYY", //adding this might match with that of MM-DD-YYYY too
	}
	for layout, format := range allDateLayouts {
		_, err = time.Parse(layout, date)
		if err == nil {
			dateFormat = format
			return
		}
	}
	return "", nil
}

func (manager *GCSManager) ListFilesWithPrefix(prefix string, maxItems int64) ([]string, error) {
	gcsObjects := []string{}
	ctx := context.Background()
	var client *storage.Client
	var err error
	if manager.Config.Credentials == "" {
		client, err = storage.NewClient(ctx)
	} else {
		client, err = storage.NewClient(ctx, option.WithCredentialsJSON([]byte(manager.Config.Credentials)))
	}

	if err != nil {
		return nil, err
	}
	it := client.Bucket(manager.Config.Bucket).Objects(ctx, &storage.Query{
		Prefix:    prefix,
		Delimiter: "",
	})
	for {
		if maxItems == 0 {break}
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Bucket(%q).Objects(): %v", manager.Config.Bucket, err)
		}
		gcsObjects = append(gcsObjects, attrs.Name)
		maxItems--
	}
	return gcsObjects, err
}

func (manager *GCSManager) getClient() (*storage.Client, error) {
	var err error
	if manager.client == nil {
		ctx := context.Background()
		manager.client, err = storage.NewClient(ctx, option.WithCredentialsJSON([]byte(manager.Config.Credentials)))
	}
	return manager.client, err
}

func (manager *GCSManager) Download(output *os.File, key string) error {
	ctx := context.Background()

	client, err := manager.getClient()

	if err != nil {
		return err
	}

	rc, err := client.Bucket(manager.Config.Bucket).Object(key).NewReader(ctx)
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(output, rc)
	return err
}

/*
GetObjectNameFromLocation gets the object name/key name from the object location url
	https://storage.googleapis.com/bucket-name/key - >> key
*/
func (manager *GCSManager) GetObjectNameFromLocation(location string) (string, error) {
	var baseURL string
	baseURL += "https://storage.googleapis.com" + "/"
	baseURL += manager.Config.Bucket + "/"
	return location[len(baseURL):], nil
}

//TODO complete this
func (manager *GCSManager) GetDownloadKeyFromFileLocation(location string) string {
	locationSlice := strings.Split(location, "storage.googleapis.com/"+manager.Config.Bucket+"/")
	pkgLogger.Debug("Location: ", location, "downloadKey: ", locationSlice[len(locationSlice)-1])
	return locationSlice[len(locationSlice)-1]
}

type GCSManager struct {
	Config *GCSConfig
	client *storage.Client
}

func GetGCSConfig(config map[string]interface{}) *GCSConfig {
	var bucketName, prefix, credentials string
	if config["bucketName"] != nil {
		bucketName = config["bucketName"].(string)
	}
	if config["prefix"] != nil {
		prefix = config["prefix"].(string)
	}
	if config["credentials"] != nil {
		credentials = config["credentials"].(string)
	}
	return &GCSConfig{Bucket: bucketName, Prefix: prefix, Credentials: credentials}
}

type GCSConfig struct {
	Bucket      string
	Prefix      string
	Credentials string
}

func (manager *GCSManager) DeleteObjects(locations []string) (err error) {
	return
}
