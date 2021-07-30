#!/bin/bash

# I will echo event location from command line args AND the env variables set from mms.
echo "product-location=$1" # Path to product dataset
echo "MMS_PRODUCT_EVENT_PRODUCT=$MMS_PRODUCT_EVENT_PRODUCT" # Json serialized mms event.
echo "MMS_PRODUCT_EVENT_PRODUCT_LOCATION=$MMS_PRODUCT_EVENT_PRODUCT_LOCATION"