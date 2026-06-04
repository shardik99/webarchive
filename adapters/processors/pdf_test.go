//go:build integration

package processors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPDF_Process(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skip test with external resource")
	}

	page := entity.NewPage("https://github.com/SebastiaanKlippert/go-wkhtmltopdf", "", nil, nil, nil, nil, 0)
	files, err := (&PDF{}).Process(context.Background(), page)
	require.NoError(t, err)
	require.Len(t, files, 1)

	f := files[0]
	fmt.Println("ID         ", f.ID)
	fmt.Println("Name       ", f.Name)
	fmt.Println("MimeType   ", f.MimeType)
	fmt.Println("Size       ", f.Size)
	fmt.Println("Created    ", f.Created.Format(time.RFC3339))
}
