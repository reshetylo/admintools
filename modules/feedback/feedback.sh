#!/bin/bash

USER=`echo $1 | base64 -d`
SUBJECT=`echo $2 | base64 -d`
FEEDBACK=`echo $3 | base64 -d`

echo -e "From: admintools-noreply@example.com\nSubject: Admintools Feedback - ${SUBJECT}\nThere is new feedback received from ${USER}\nMessage content:\n${FEEDBACK}\n\nRegards,\nAdmintools" | sendmail yurii.reshetylo@gmail.com
responsecode=$?
if [ "$responsecode" -gt 0 ]; then
    echo -e "Something went wrong with sending. Maybe your message body is too long?\n\nSendmail error code = $responsecode"
else
    echo "Your feedback sent. Thanks"
fi