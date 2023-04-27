package main

import (
	"bytes"
	"encoding/base64"
	json "encoding/json"
	"fmt"
	pdf "github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/valyala/fasthttp"
	"html/template"
	"log"
	"os"
	"time"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

func main() {
	// get port from env
	port := os.Getenv("PORT")
	host := os.Getenv("HOST")
	addr := fmt.Sprintf("%s:%s", host, port)
	log.Printf("Listen address: %s", addr)

	loadTemplates()

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		log.Printf("request{url=%s}", ctx.Path())
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
		log.Printf("request{url=%s, elapsed=%s}", path, elapsed)
	}

	if err := fasthttp.ListenAndServe(addr, requestHandler); err != nil {
		log.Fatalf("error in ListenAndServe: %v", err)
	}
}

func loadTemplates() {

}

func templateHandle(ctx *fasthttp.RequestCtx) {
	log.Println("start template handle")
	if ctx.IsPost() == false {
		log.Printf("method not allowed")
		ctx.Error("", fasthttp.StatusMethodNotAllowed)
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		log.Printf("error while parse multipart form: %v", err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}

	if len(form.Value["template"]) == 0 || len(form.File["file"]) == 0 {
		log.Printf("template or file not found")
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	templateId := form.Value["template"][0]
	log.Printf("register template: %s", templateId)

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
		log.Printf("error while read file: %v", err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}

	if templates.Lookup(templateId) != nil {
		log.Printf("template already exist")
		response := BaseResponse{
			Code:    "01",
			Message: "template already exist",
		}
		jsonResponse, _ := json.Marshal(response)
		ctx.Error(string(jsonResponse), fasthttp.StatusBadRequest)
		return
	}

	tpl, err := template.New(templateId).Parse(string(templateBytes))
	if err != nil {
		log.Printf("error while parse template: %v", err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}
	_, err = templates.AddParseTree(templateId, tpl.Tree)
	if err != nil {
		log.Printf("error while add template: %v", err)
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

func exportPdf(ctx *fasthttp.RequestCtx) {
	log.Println("start export pdf")
	if ctx.IsPost() == false {
		log.Printf("method not allowed")
		ctx.Error("", fasthttp.StatusMethodNotAllowed)
		return
	}

	var req ExportRequest
	err := json.Unmarshal(ctx.PostBody(), &req)
	if err != nil {
		log.Printf("error while unmarshal: %v", err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}

	log.Printf("request: %v", req)

	tpl := templates.Lookup(req.Template)
	if tpl == nil {
		log.Printf("template not found")
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	// map data to template
	var val bytes.Buffer
	if err := tpl.Execute(&val, req.Data); err != nil {
		log.Printf("error while execute template: %v", err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}

	// generate pdf
	pdfdata, err := genPdfFromHtml(val.Bytes())
	if err != nil {
		log.Printf("error: %v", err)
		ctx.Error("", fasthttp.StatusInternalServerError)
		return
	}

	res := BaseResponse{
		Code: "00",
		Data: base64.StdEncoding.EncodeToString(pdfdata),
	}
	responseBytes, err := json.Marshal(res)
	if err != nil {
		log.Printf("error while marshal: %v", err)
		ctx.Error("", fasthttp.StatusInternalServerError)
	}
	_, err = ctx.Write(responseBytes)
	if err != nil {
		log.Fatal(err)
	}
}

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

func reloadTemplate(loader TemplateLoader) {
	allTemplates, err := loader.LoadAllTemplates()
	if err != nil {
		log.Printf("error while load all templates: %v", err)
		return
	}

	for key, value := range allTemplates {
		log.Printf("register template: %s", key)

		tpl := templates.Lookup(key)
		if tpl != nil {
			continue
		}
		tpl, err := template.New(key).Parse(value)
		if err != nil {
			log.Printf("error while parse template: %v", err)
			return
		}
		_, err = templates.AddParseTree(key, tpl.Tree)
		if err != nil {
			log.Printf("error while add template: %v", err)
			return
		}
	}
}
