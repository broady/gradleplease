# Extract a URL from addon.xml
function extract_url {
  EXTRACT_URL=$(grep $@ addon.xml | sed -e 's/[^>]*>\([^<]*\).*/\1/')
}

BASE_URL=http://dl-ssl.google.com/android/repository
curl $BASE_URL/addon.xml > addon.xml

extract_url google_m2repository
curl "$BASE_URL/$EXTRACT_URL" > google_m2repository.zip
zipinfo google_m2repository.zip | grep pom$ | grep play-services-base
rm google_m2repository.zip

extract_url android_m2repository
curl "$BASE_URL/$EXTRACT_URL" > android_m2repository.zip
zipinfo android_m2repository.zip | grep pom$ | grep support-v4
rm android_m2repository.zip

rm addon.xml
