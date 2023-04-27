curl --location 'http://localhost:8888' \
--header 'Content-Type: application/json' \
--data '{
    "template": "T1",
    "data": {
        "seller": {
            "name": "Cong ty ban hang",
            "taxcode": "13213123123"
        },
        "buyer": {
            "company_name": "VNPAY",
            "name": "Nguyen quang truong",
            "address": "22 Lang ha"
        }
    }
}'