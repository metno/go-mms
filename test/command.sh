#!/bin/bash

# I will echo event location from command line args AND the env variables set from mms.
echo "product-location=$1" # Path to product dataset
echo "MMS_EVENT=$MMS_EVENT" # Json serialized mms event.