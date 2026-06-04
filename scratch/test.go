package main
import (
    "context"
    "fmt"
    "os"
    "github.com/sethvargo/go-envconfig"
)
type Config struct {
    UI UI `env:",prefix=UI_"`
}
type UI struct {
    Theme string `env:"THEME,default=basic"`
}
func main() {
    os.Setenv("UI_THEME", "dark")
    var cfg Config
    envconfig.ProcessWith(context.Background(), &envconfig.Config{Target: &cfg, Lookuper: envconfig.OsLookuper()})
    fmt.Println("Theme is:", cfg.UI.Theme)
}
