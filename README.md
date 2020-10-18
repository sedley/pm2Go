# AQI status webservice

This service displays the current AQI for the Redwood City area in a 500px by 500px png at `/image.png`, a json payload is
also available at `/aqi`.

## Deployment
This service is intended to be deployed on cloud run using cloud build.  This can be done in two steps assuming you've already
installed the gcloud sdk and set up a project. Assuming your gcloud project id is `abc-123`, run:
1. `gcloud builds submit --tag gcr.io/abc-123/pm2go`
2. `gcloud run deploy --image gcr.io/abc-123/pm2go --platform managed`

After these complete you should map a domain to your new service.
