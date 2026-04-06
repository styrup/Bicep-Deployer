package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// BlobClient wraps Azure Blob Storage operations for Bicep templates.
type BlobClient struct {
	client        *azblob.Client
	containerName string
}

// New creates a BlobClient. It uses the connection string when provided,
// otherwise falls back to DefaultAzureCredential (Managed Identity / az login).
func New(accountName, containerName, connectionString string) (*BlobClient, error) {
	var client *azblob.Client
	var err error

	if connectionString != "" {
		client, err = azblob.NewClientFromConnectionString(connectionString, nil)
	} else {
		cred, credErr := azidentity.NewDefaultAzureCredential(nil)
		if credErr != nil {
			return nil, fmt.Errorf("DefaultAzureCredential: %w", credErr)
		}
		serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net", accountName)
		client, err = azblob.NewClient(serviceURL, cred, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("create blob client: %w", err)
	}

	return &BlobClient{client: client, containerName: containerName}, nil
}

// ListTemplates returns the names of all .bicep files in the container.
func (b *BlobClient) ListTemplates(ctx context.Context) ([]string, error) {
	var names []string

	pager := b.client.NewListBlobsFlatPager(b.containerName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list blobs: %w", err)
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name != nil && strings.HasSuffix(*item.Name, ".bicep") {
				names = append(names, *item.Name)
			}
		}
	}

	return names, nil
}

// DownloadTemplate downloads and returns the content of a named .bicep file.
func (b *BlobClient) DownloadTemplate(ctx context.Context, name string) (string, error) {
	resp, err := b.client.DownloadStream(ctx, b.containerName, name, nil)
	if err != nil {
		return "", fmt.Errorf("download blob %q: %w", name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read blob %q: %w", name, err)
	}

	return string(data), nil
}
