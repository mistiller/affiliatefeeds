# Affiliate Feeds

## ! Decommissioned !

This is a decommissioned older version of the feed engine we have built to deliver product feeds to our websites. It has been mostly rewritten in recent weeks and months which is why this version largely serves as documentation and a scaffolding for the new iteration to be based on.

### Sources:
    - Tradedoubler affiliate product feeds
    - Awin Affilate product feeds
    - Various website crawler examples

### Destinations:
    - Wordpress / Woocommerce API upload
    - Vue Storefront feed upload (experimental)
    - CSV export

### Features:
    - On-disk caching via Badger-DB
    - Additional Product Data via Dynamo-DB connector
    - Live Mapping Tables for term translation via Google Sheets
    - Exemplary deployment scripts using AWS ECR and VPC 

