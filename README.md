# aws-rekognition-sandbox
Amazon Rekognitionで出来る事を調査する為の検証用プロジェクト

## Getting Started

AWS Lambda + Goで実装しています。

デプロイには [serverless framework](https://www.serverless.com/) を利用しています。

### AWSクレデンシャルの設定

[名前付きプロファイル](https://docs.aws.amazon.com/ja_jp/cli/latest/userguide/cli-configure-profiles.html) を利用しています。

このプロジェクトで利用しているプロファイル名は `nekochans-dev` です。

### 環境変数の設定

[direnv](https://github.com/direnv/direnv) 等を利用して環境変数を設定します。

```
export DEPLOY_STAGE=デプロイステージを設定、デフォルトは dev
export REGION=AWSのリージョンを指定、例えば ap-northeast-1 等
export TRIGGER_BUCKET_NAME=Lambda関数実行のトリガーとなるS3バケット名を指定
```

### デプロイ

1. `npm ci` を実行（初回のみでOK）
1. `make deploy` を実行

## Lambda関数の仕様

### imageRecognition

Amazon Rekognitionで取得出来るラベルをそのまま返すAPIです。

例えば `test/images/abyssinian-cat.jpg` を解析したい場合は以下のように実行します。

```
# MacOS上からzshシェルを用いて実行しています
echo '{"image" : "'"$( base64 ./test/images/abyssinian-cat.jpg)"'"}' | \
curl -v -X POST -H "Content-Type: application/json" -d @- https://YOUR_APIID.execute-api.ap-northeast-1.amazonaws.com/images/recognition | jq
```

下記のようなレスポンスが返ってきます。

```json
{
  "labels": [
    {
      "Confidence": 98.68521118164062,
      "Instances": [
        {
          "BoundingBox": {
            "Height": 0.8715125322341919,
            "Left": 0.01610049419105053,
            "Top": 0.0782160758972168,
            "Width": 0.9815520644187927
          },
          "Confidence": 98.68521118164062
        }
      ],
      "Name": "Cat",
      "Parents": [
        {
          "Name": "Pet"
        },
        {
          "Name": "Mammal"
        },
        {
          "Name": "Animal"
        }
      ]
    },
    {
      "Confidence": 98.68521118164062,
      "Instances": [],
      "Name": "Pet",
      "Parents": [
        {
          "Name": "Animal"
        }
      ]
    },
    {
      "Confidence": 98.68521118164062,
      "Instances": [],
      "Name": "Mammal",
      "Parents": [
        {
          "Name": "Animal"
        }
      ]
    },
    {
      "Confidence": 98.68521118164062,
      "Instances": [],
      "Name": "Animal",
      "Parents": []
    },
    {
      "Confidence": 95.80082702636719,
      "Instances": [],
      "Name": "Abyssinian",
      "Parents": [
        {
          "Name": "Cat"
        },
        {
          "Name": "Pet"
        },
        {
          "Name": "Mammal"
        },
        {
          "Name": "Animal"
        }
      ]
    }
  ]
}
```

サンプルコードなので `.jpg` 以外の画像は受け付けていません。

### judgeIfCatImage

`TRIGGER_BUCKET_NAME` で指定したS3バケットの `tmp/` フォルダに `.jpg` のファイルがアップロードされた場合に起動します。

画像が🐱の画像かどうかを判定し、🐱画像だった場合は `TRIGGER_BUCKET_NAME` の `cat-images/` フォルダに移動させます。

`imageRecognition` をコールすると `TRIGGER_BUCKET_NAME` で指定したS3バケットの `tmp/` フォルダに画像が入るので、それで動作確認が可能です。

ちなみに本プロジェクトでは活用していませんが、以下のように内部処理で🐱の種類（マンチカン、スコティッシュフォールドとか）を画像の解析結果から判定しています。

これらをDB等に保存しておけば、画像検索の要素として使えるかもしれません。

- `test/images/abyssinian-cat.jpg`の場合は以下のようになる

`{"isCatImage": true, "typesOfCats": ["Abyssinian"]}`

- `test/images/manx-cat.jpg` の場合は以下のようになる

`{"isCatImage": true, "typesOfCats": ["Manx"]}`
