package bid

// BidResponse3 fixture based on https://github.com/InteractiveAdvertisingBureau/openrtb for the OpenRTB 3.0
const BidResponse3 = `{
  "openrtb": {
    "ver": "3.0",
    "domainspec": "adcom",
    "domainver": "1.0",
    "response": {
      "id": "0123456789ABCDEF",
      "bidid": "0011223344AABBCC",
      "seatbid": [
        {
          "seat": "XYZ",
          "bid": [
            {
              "id": "yaddayadda",
              "item": "1",
              "price": 1.5,
              "deal": "1234",
              "tactic": "...",
              "purl": "...",
              "burl": "...",
              "lurl": "...",
              "mid": "...",
              "macro": [
                {
                  "key": "TIMESTAMP",
                  "value": "1127987134"
                },
                {
                  "key": "CLICKTOKEN",
                  "value": "A7D800F2716DB"
                }
              ],
              "media": {
                "ad": {
                  "id": "d0bcb39723af87c2bb00942afee5710e",
                  "adomain": [
                    "ford.com"
                  ],
                  "secure": 1,
                  "display": {
                    "mime": "image/jpeg",
                    "ctype": 3,
                    "w": 320,
                    "h": 50,
                    "banner": {
                      "img": "https://somebuyer.test/creative",
                      "link": {
                        "url": "https://somebuyer.test/click",
                        "urlfb": "https://somebuyer.test"
                      }
                    },
                    "event": [
                      {
                        "type": 1,
                        "method": 1,
                        "url": "https://somebuyer.test/pixel"
                      }
                    ]
                  }
                }
              }
            }
          ]
        }
      ]
    }
  }
}`

// BidResponse2 fixture based on BidResponse3 but for OpenRTB 2.5
const BidResponse2 = `{
  "id":"0123456789ABCDEF",
  "seatbid":[
    {
      "bid":[
        {
          "id":"yaddayadda",
          "impid":"1",
          "price":1.5,
          "nurl":"...",
          "burl":"...",
          "lurl":"...",
          "adid":"...",
          "adomain":[
            "ford.com"
          ],
          "crid":"d0bcb39723af87c2bb00942afee5710e",
          "tactic":"...",
          "dealid":"1234",
          "w":320,
          "h":50,
          "ext":{
            "macro":[
              {
                "key":"TIMESTAMP",
                "value":"1127987134"
              },
              {
                "key":"CLICKTOKEN",
                "value":"A7D800F2716DB"
              }
            ],
            "secure":1,
            "mime":"image/jpeg",
            "ctype":3,
            "event":[
              {
                "type":1,
                "method":1,
                "url":"https://somebuyer.test/pixel"
              }
            ],
            "banner":{
              "img":"https://somebuyer.test/creative",
              "link":{
                "url":"https://somebuyer.test/click",
                "urlfb":"https://someburer.test"
              }
            }
          }
        }
      ],
      "seat":"XYZ"
    }
  ],
  "bidid":"0011223344AABBCC"
}`
