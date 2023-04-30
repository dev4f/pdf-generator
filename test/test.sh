curl --location 'http://localhost:8888' \
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