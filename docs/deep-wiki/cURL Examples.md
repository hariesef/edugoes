# cURL Examples

Keywords: curl, NRPS, AGS, lineitems, scores, results, Authorization Bearer, Content-Type application/json

## NRPS list
```bash
curl -H "Authorization: Bearer <token>" \
  "https://<host>/api/nrps/contexts/dev-context/members?limit=50&offset=0"
```

## AGS create line item
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"label":"Quiz 1","scoreMaximum":100,"resourceLinkId":"<resource_link_id>"}' \
  "https://<host>/api/ags/contexts/dev-context/lineitems"
```

## AGS post score
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"userId":"student@efrika.net","scoreGiven":85,"activityProgress":"Completed","gradingProgress":"FullyGraded"}' \
  "https://<host>/api/ags/contexts/dev-context/lineitems/<id>/scores"
```
