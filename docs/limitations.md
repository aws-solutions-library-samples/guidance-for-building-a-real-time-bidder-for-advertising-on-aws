---
title: Limitations
---

This documents lists expected features of a bidder in a demand side platform that are incomplete or missing in the
implemented application.

A DSP requires multiple additional components. Here we assume that they already exist and have complete functionality.

## Bid request format

Only OpenRTB 3.0 bid requests are supported. JSON is used for them: an HTTP request body contains a JSON object with
the `openrtb` property containing the top-level
[Openrtb](https://github.com/InteractiveAdvertisingBureau/openrtb/blob/master/OpenRTB%20v3.0%20FINAL.md#object_openrtb)
object.

There is no support for compression: gzip-encoded bid requests require special support from a proxy before arriving in
the bidder.

Validation of bid requests is very minimal: any valid JSON object with non-empty values in the required fields is
accepted.

## Logging

All valid bid requests are logged to the data stream. There is no filtering of possibly fraudulent, restricted by
regulations or not useful bid requests.

All bid responses are logged to the same data stream as bid requests. This might be sufficient, unless different
retention policies or scaling is needed to process both kinds of data.

## Targeting

Targeting is done only by device ID. There are no checks if the matched campaign supports the bid request placement or
if the publisher disallows that campaign (by e.g. blocked categories or domains).

## Auction

An inner first price auction occurs: if multiple campaigns have the same price, there is no effort to optimize total
platform spends or deliver both campaigns. If a campaign has a lower price than another matching campaign, it's
unlikely to spend its whole budget.

Only the USD currency is supported. Bid floors and auction type are not inspected by the bidder.

There is no support for private marketplace deals.

Pricing models other than CPM and impression multipliers are not supported.

## Bid response

Responses use the
[Openrtb](https://github.com/InteractiveAdvertisingBureau/openrtb/blob/master/OpenRTB%20v3.0%20FINAL.md#object_openrtb)
wrapper object.

A placeholder creative specification is included in the bid response: any production integration would need additional
fields in the campaign cache for specifying creative metadata and tracking URLs. Some fields recommended by OpenRTB and
required by exchanges might be missing.

Parameters like the billing URL are placeholders of a realistic size; they need to point to a real tracker.

Seat ID is fixed and there is no support for multiple bids of different seats in the same bid response.
