# rest/db-browse-get.md

{% api-method method="get" host="" path="/rest/db/browse" %}
{% api-method-summary %}

{% endapi-method-summary %}

{% api-method-description %}

{% endapi-method-description %}

{% api-method-spec %}
{% api-method-request %}
{% api-method-query-parameters %}
{% api-method-parameter name="prefix" type="string" required=false %}
Path prefix
{% endapi-method-parameter %}

{% api-method-parameter name="levels" type="integer" required=false %}
Number of levels to recurse
{% endapi-method-parameter %}

{% api-method-parameter name="folder" type="string" required=true %}
Folder ID
{% endapi-method-parameter %}
{% endapi-method-query-parameters %}
{% endapi-method-request %}

{% api-method-response %}
{% api-method-response-example httpCode=200 %}
{% api-method-response-example-description %}

{% endapi-method-response-example-description %}

```javascript
{
   "directory" : {
      "file" : [
         "2015-04-20T22:20:45+09:00",
         130940928
      ],
      "subdirectory" : {
         "another file" : [
            "2015-04-20T22:20:45+09:00",
            130940928
         ]
      }
   },
   "rootfile" : [
      "2015-04-20T22:20:45+09:00",
      130940928
   ]
}
```
{% endapi-method-response-example %}
{% endapi-method-response %}
{% endapi-method-spec %}
{% endapi-method %}

Returns the directory tree of the global model. Directories are always JSON objects \(map/dictionary\), and files are always arrays of modification time and size. The first integer is the files modification time, and the second integer is the file size.

The call takes one mandatory `folder` parameter and two optional parameters. Optional parameter `levels` defines how deep within the tree we want to dwell down \(0 based, defaults to unlimited depth\) Optional parameter `prefix` defines a prefix within the tree where to start building the structure.

{% hint style="danger" %}
This is an expensive call, increasing CPU and RAM usage on the device. Use sparingly.
{% endhint %}

