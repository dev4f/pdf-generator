# Export a PDF file from a template
curl --location 'http://localhost:8888/export' \
--header 'Content-Type: application/json' \
--data '{
    "template": "T1",
    "data": {
        "seller": {
            "name": "John Doe",
            "taxcode": "1234567890",
        }
    }
}'

# Import a template
curl -v \                                                                                                 ✔
-F "template=T1" \
-F "file=@/home/template.html" \
http://localhost:8888/templates