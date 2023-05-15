# A simple service to export pdf by html template

## Dependencies
- [wkhtmltopdf](https://wkhtmltopdf.org/downloads.html)
## How to build
- Clone this repo
- Run 
```
docker build -t pdf-exporter .
docker run -rm -p 8888:8888 pdf-exporter
```

## How to use
- Config the environment variables
  - `SERVER_PORT`: Port to run the service
  - `SERVER_HOST`: Host to run the service
  - `STORAGE_TYPE`: Type of storage to store and load templates, 0: inmemory, 1: minio
  - `STORAGE_MINIO_ENDPOINT`: Minio endpoint
  - `STORAGE_MINIO_ACCESS_KEY`: Minio access key
  - `STORAGE_MINIO_SECRET_KEY`: Minio secret key
  - `STORAGE_MINIO_BUCKET_NAME`: Minio bucket name
  - `STORAGE_MINIO_PATH`: Path to store templates in minio
  - `TEMPLATE_NORMALIZATION_FORM`: Normalization form to normalize unicode characters, 0:NFC, 1:NFD, 2:NFKC, 3:NFKD , see [Normalization Form](https://go.dev/blog/normalization)
### How to export pdf
- Send a POST request to http://localhost:8888/export with body
```
{
    "template": "Template name",
    "data": {
        "name": "John Doe"
    }
}
```

### How to add new template
- Add new template to `templates` folder
- Add new template by send a POST request to http://localhost:8888/templates see api.yaml and test/test.sh to get more details

### Template engine
- [Go template](https://pkg.go.dev/html/template)

For example:

Template
```
<html>
    <head>
        <title>Hello {{.Test.Title}}</title>
    </head>
    <body>
        <h1>Hello {{.Test.Name}}</h1>
    </body>
</html>
```

and data
```
{
    "Test": {
        "Title": "World",
        "Name": "John Doe"
    }
}
```

will be rendered to
```
<html>
    <head>
        <title>Hello World</title>
    </head>
    <body>
        <h1>Hello John Doe</h1>
    </body>
</html>
```