package main
import ("fmt";"log";"net/http";"os";"github.com/stockyard-dev/stockyard-fence/internal/server";"github.com/stockyard-dev/stockyard-fence/internal/store")
func main(){port:=os.Getenv("PORT");if port==""{port="8640"};dataDir:=os.Getenv("DATA_DIR");if dataDir==""{dataDir="./fence-data"}
db,err:=store.Open(dataDir);if err!=nil{log.Fatalf("fence: %v",err)};defer db.Close();srv:=server.New(db)
fmt.Printf("\n  Fence — API key vault\n  Dashboard:  http://localhost:%s/ui\n  API:        http://localhost:%s/api\n\n",port,port)
log.Printf("fence: listening on :%s",port);log.Fatal(http.ListenAndServe(":"+port,srv))}
