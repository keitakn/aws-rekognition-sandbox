# aws-rekognition-sandbox
[![ci](https://github.com/keitakn/aws-rekognition-sandbox/actions/workflows/ci.yml/badge.svg)](https://github.com/keitakn/aws-rekognition-sandbox/actions/workflows/ci.yml)
[![cd](https://github.com/keitakn/aws-rekognition-sandbox/actions/workflows/cd.yml/badge.svg)](https://github.com/keitakn/aws-rekognition-sandbox/actions/workflows/cd.yml)

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
echo '{"image" : "'"$( base64 ./test/images/abyssinian-cat.jpg)"'", "imageExtension": ".jpg"}' | \
curl -v -X POST -H "Content-Type: application/json" -d @- https://xxxxxxxxxx.execute-api.ap-northeast-1.amazonaws.com/images/recognition | jq
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

`.jpg`, `.jpeg`, `.png`, `.webp` 以外の画像は受け付けていません。

### detectFaces

Amazon Rekognition [イメージ内の顔の検出API](https://docs.aws.amazon.com/ja_jp/rekognition/latest/dg/faces-detect-images.html) で取得出来るラベルをそのまま返すAPIです。

以下のようにリクエストを行います。

```
# MacOS上からzshシェルを用いて実行しています
echo '{"image" : "'"$( base64 ./test/images/manx-cat.jpg)"'"}' | \
curl -v -X POST -H "Content-Type: application/json" -d @- https://xxxxxxxxxx.execute-api.ap-northeast-1.amazonaws.com/images/faces | jq
```

下記のようなレスポンスが返ってきます。

```json
{
  "faceDetails": [
    {
      "AgeRange": null,
      "Beard": null,
      "BoundingBox": {
        "Height": 0.29183787,
        "Left": 0.07472489,
        "Top": 0.2709639,
        "Width": 0.5417548
      },
      "Confidence": 66.72364,
      "Emotions": null,
      "Eyeglasses": null,
      "EyesOpen": null,
      "Gender": null,
      "Landmarks": [
        {
          "Type": "eyeLeft",
          "X": 0.30981517,
          "Y": 0.33770314
        },
        {
          "Type": "eyeRight",
          "X": 0.50000805,
          "Y": 0.33927107
        },
        {
          "Type": "mouthLeft",
          "X": 0.30706555,
          "Y": 0.50413615
        },
        {
          "Type": "mouthRight",
          "X": 0.46517605,
          "Y": 0.5042505
        },
        {
          "Type": "nose",
          "X": 0.47034857,
          "Y": 0.42010358
        }
      ],
      "MouthOpen": null,
      "Mustache": null,
      "Pose": {
        "Pitch": 12.441085,
        "Roll": 9.164827,
        "Yaw": 6.8097568
      },
      "Quality": {
        "Brightness": 95.53643,
        "Sharpness": 92.22801
      },
      "Smile": null,
      "Sunglasses": null
    },
    {
      "AgeRange": null,
      "Beard": null,
      "BoundingBox": {
        "Height": 0.2223244,
        "Left": 0.7428785,
        "Top": 0.74860626,
        "Width": 0.3278522
      },
      "Confidence": 66.78572,
      "Emotions": null,
      "Eyeglasses": null,
      "EyesOpen": null,
      "Gender": null,
      "Landmarks": [
        {
          "Type": "eyeLeft",
          "X": 0.86015856,
          "Y": 0.7967467
        },
        {
          "Type": "eyeRight",
          "X": 0.91044015,
          "Y": 0.82621884
        },
        {
          "Type": "mouthLeft",
          "X": 0.7535296,
          "Y": 0.8594641
        },
        {
          "Type": "mouthRight",
          "X": 0.7945388,
          "Y": 0.8837279
        },
        {
          "Type": "nose",
          "X": 0.81529987,
          "Y": 0.8392649
        }
      ],
      "MouthOpen": null,
      "Mustache": null,
      "Pose": {
        "Pitch": 9.174266,
        "Roll": 60.05953,
        "Yaw": 24.53278
      },
      "Quality": {
        "Brightness": 85.71859,
        "Sharpness": 4.374837
      },
      "Smile": null,
      "Sunglasses": null
    }
  ]
}
```

人の顔がはっきり写っている場合は、信頼度（Confidence）はかなり高めに出ます。（99%以上が多かったです）

しかし動物の顔を検出する事もあります。（その場合は信頼度（Confidence）は低めになります。）

### judgeIfCatImage

`TRIGGER_BUCKET_NAME` で指定したS3バケットの `tmp/` フォルダにファイルがアップロードされた場合に起動します。

画像が🐱の画像かどうかを判定し、🐱画像だった場合は `TRIGGER_BUCKET_NAME` の `cat-images/` フォルダに移動させます。

`imageRecognition` をコールすると `TRIGGER_BUCKET_NAME` で指定したS3バケットの `tmp/` フォルダに画像が入るので、それで動作確認が可能です。

ちなみに本プロジェクトでは活用していませんが、以下のように内部処理で🐱の種類（マンチカン、スコティッシュフォールドとか）を画像の解析結果から判定しています。

これらをDB等に保存しておけば、画像検索の要素として使えるかもしれません。

- `test/images/abyssinian-cat.jpg`の場合は以下のようになる

`{"isCatImage": true, "typesOfCats": ["Abyssinian"]}`

- `test/images/manx-cat.jpg` の場合は以下のようになる

`{"isCatImage": true, "typesOfCats": ["Manx"]}`

## テストコードの作成

テストコードは `aws-sdk-go-v2` をモックに置き換える形で実装します。

下記のようにモックを生成します。（GoのDockerコンテナの中で実行します）

```bash
mockgen -source=infrastructure/rekognition_client.go -destination mock/rekognition_client.go -package mock
```

こちらのコマンドは `mock/rekognition_client.go` を生成した時のものです。

他にもモックが必要な物があればこちらと同じようにモック化します。

モックを生成する際は以下のルールに従って生成します。

- package名は `mock`
- `mock/` ディレクトリに配置する
