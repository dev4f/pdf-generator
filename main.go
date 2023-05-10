package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	pdf "github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/bluele/gcache"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"html/template"
	"log"
	"strings"
	"time"
)

var templateManager TemplateManager

var templates gcache.Cache

func main() {

	config, err := loadConfig()
	if err != nil {
		log.Printf("error while load config: %v", err)
		return
	}

	addr := fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port)
	log.Printf("Listen address: %s", addr)

	templateManager, err = createTemplateManager(config)
	if err != nil {
		log.Fatalf("error while init template manager: %v", err)
		return
	}
	loadTemplate := func(key interface{}) (interface{}, error) {
		tplString, err := templateManager.GetByName(key.(string))
		if err != nil {
			return nil, err
		}
		if tplString == "" {
			return nil, fmt.Errorf("template %s not found", key.(string))
		}
		tpl, err := template.New(key.(string)).Parse(tplString)
		if err != nil {
			return nil, err
		}
		return tpl, nil
	}
	templates = gcache.New(100).LRU().LoaderFunc(loadTemplate).Expiration(10 * time.Minute).Build()

	if err := fasthttp.ListenAndServe(addr, handle); err != nil {
		log.Fatalf("error in ListenAndServe: %v", err)
	}
}

func loadConfig() (*Config, error) {

	var config Config
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	envKeysMap := map[string]interface{}{}
	if err := mapstructure.Decode(config, &envKeysMap); err != nil {
		return nil, err
	}
	structKeys := flattenAndMergeMap(map[string]bool{}, envKeysMap, "")
	for key, _ := range structKeys {
		if err := viper.BindEnv(key); err != nil {
			return nil, err
		}
	}
	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func createTemplateManager(config *Config) (TemplateManager, error) {
	if config.Storage.Type == StorageTypeInMemory {
		return NewInMemoryTemplateManager(), nil
	} else if config.Storage.Type == StorageTypeMinio {
		return NewMinioTemplateManager(
			config.Storage.Minio.Endpoint,
			config.Storage.Minio.AccessKey,
			config.Storage.Minio.SecretKey,
			config.Storage.Minio.UseSSL,
			config.Storage.Minio.Bucket,
			config.Storage.Minio.Path,
		)
	} else {
		return nil, fmt.Errorf("unknown template manager type: %s", config.Storage.Type)
	}
}

// handle is the main handler for all requests
func handle(ctx *fasthttp.RequestCtx) {
	ctx.SetUserValue("traceId", uuid.New().String())
	start := time.Now()
	path := string(ctx.Path())
	if path == "/health" {
		health(ctx)
	} else if path == "/export" {
		exportPdf(ctx)
	} else if path == "/templates" {
		templateHandle(ctx)
	} else {
		ctx.Error("", fasthttp.StatusNotFound)
	}
	elapsed := time.Since(start)
	log.Printf("[%s] request{url=%s, elapsed=%s}", getTraceId(ctx), path, elapsed)
}

// templateHandle is the handler for all requests to /templates
func templateHandle(ctx *fasthttp.RequestCtx) {
	log.Printf("[%s] start template handle", getTraceId(ctx))
	if ctx.IsPost() == false {
		log.Printf("[%s] method not allowed", getTraceId(ctx))
		ctx.Error("", fasthttp.StatusMethodNotAllowed)
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		log.Printf("[%s] error while parse multipart form: %v", getTraceId(ctx), err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}

	if len(form.Value["template"]) == 0 || len(form.File["file"]) == 0 {
		log.Printf("[%s] template or file not found", getTraceId(ctx))
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	templateId := form.Value["template"][0]
	log.Printf("[%s] templateId: %s", getTraceId(ctx), templateId)

	templateFile, err := form.File["file"][0].Open()
	if err != nil {
		log.Printf("error while open file: %v", err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}
	defer templateFile.Close()

	templateBytes := make([]byte, form.File["file"][0].Size)
	_, err = templateFile.Read(templateBytes)
	if err != nil {
		log.Printf("[%s] error while read file: %v", getTraceId(ctx), err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}

	if tpl, err := templates.Get(templateId); tpl != nil && err == nil {
		log.Printf("[%s] template already exist", getTraceId(ctx))
		response := BaseResponse{
			Code:    "01",
			Message: "template already exist",
		}
		jsonResponse, _ := json.Marshal(response)
		ctx.Error(string(jsonResponse), fasthttp.StatusBadRequest)
		return
	}
	err = templateManager.Add(templateId, string(templateBytes))
	if err != nil {
		log.Printf("[%s] error while add template: %v", getTraceId(ctx), err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}
	ctx.SetStatusCode(fasthttp.StatusCreated)
}

func health(ctx *fasthttp.RequestCtx) {
	_, err := ctx.Write([]byte("ok"))
	if err != nil {
		log.Fatal(err)
	}
}

// exportPdf is the handler for all requests to /export
func exportPdf(ctx *fasthttp.RequestCtx) {
	log.Printf("[%s] start export pdf", getTraceId(ctx))
	if ctx.IsPost() == false {
		log.Printf("method not allowed")
		ctx.Error("", fasthttp.StatusMethodNotAllowed)
		return
	}

	log.Printf("[%s] request body: %s", getTraceId(ctx), string(ctx.PostBody()))
	var req ExportRequest
	err := json.Unmarshal(ctx.PostBody(), &req)
	if err != nil {
		log.Printf("[%s] error while parse request body: %v", getTraceId(ctx), err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}

	tplObject, err := templates.Get(req.Template)
	if err != nil {
		log.Printf("[%s] error while get template: %v", getTraceId(ctx), err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}

	if tplObject == nil {
		log.Printf("[%s] template not found", getTraceId(ctx))
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	// cast interface to template
	tpl, ok := tplObject.(*template.Template)
	if !ok {
		log.Printf("[%s] error while cast template", getTraceId(ctx))
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}

	// map data to template
	var val bytes.Buffer
	if err := tpl.Execute(&val, req.Data); err != nil {
		log.Printf("[%s] error while execute template: %v", getTraceId(ctx), err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}

	// generate pdf
	pdfdata, err := genPdfFromHtml(val.Bytes())
	if err != nil {
		log.Printf("[%s] error while generate pdf: %v", getTraceId(ctx), err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}
	ctx.Response.Header.Set("Content-Type", "application/pdf")
	ctx.Request.Header.Peek("Content-Type")
	ctx.Response.Header.Set("Content-Disposition", "attachment; filename="+req.FileName)
	_, err = ctx.Write(pdfdata)
	if err != nil {
		log.Fatalf("[%s] error while write response: %v", getTraceId(ctx), err)
	}
}

// genPdfFromHtml generate pdf from html bytes
func genPdfFromHtml(html []byte) ([]byte, error) {
	pdfg, err := pdf.NewPDFGenerator()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	// Add one page from html bytes
	pdfg.AddPage(pdf.NewPageReader(bytes.NewReader(html)))

	// Create PDF document in internal buffer
	err = pdfg.Create()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return pdfg.Bytes(), nil
}

// getTraceId get traceId from request context
func getTraceId(ctx *fasthttp.RequestCtx) string {
	return ctx.UserValue("traceId").(string)
}

func flattenAndMergeMap(shadow map[string]bool, m map[string]interface{}, prefix string) map[string]bool {
	if shadow != nil && prefix != "" && shadow[prefix] {
		// prefix is shadowed => nothing more to flatten
		return shadow
	}
	if shadow == nil {
		shadow = make(map[string]bool)
	}

	var m2 map[string]interface{}
	if prefix != "" {
		prefix += "."
	}
	for k, val := range m {
		fullKey := prefix + k
		switch val.(type) {
		case map[string]interface{}:
			m2 = val.(map[string]interface{})
		case map[interface{}]interface{}:
			m2 = cast.ToStringMap(val)
		default:
			// immediate value
			shadow[strings.ToLower(fullKey)] = true
			continue
		}
		// recursively merge to shadow map
		shadow = flattenAndMergeMap(shadow, m2, fullKey)
	}
	return shadow
}
