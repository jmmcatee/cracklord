var cracklord = angular.module('cracklord', [
	'ui.router',
   'ui.sortable',
   'ngCookies',
   'ngResource',
   'schemaForm',
   'angular-growl',
   'ngAnimate',
   'ngCsv',
   'relativeDate',
   'chart.js',
   'angularSchemaFormBase64FileUpload'
]);

cracklord.config(['growlProvider', function ($growlProvider) {
  $growlProvider.globalTimeToLive(5000);
//  growlProvider.globalDisableCountDown(true);
}]);

cracklord.config(function(base64FileUploadConfigProvider) {
	console.log(base64FileUploadConfigProvider);
	base64FileUploadConfigProvider.setDropText('New');
});

function sortEnumInJSON(input) {
   for(var key in input) {
      if(!input.hasOwnProperty(key)) continue;

      var value = input[key];
      var type = Object.prototype.toString.call(value);
      if((key === 'enum') && (type === '[object Array]')) {
         value.sort();
      } else if (typeof value === "object") {
         sortEnumInJSON(value);
      }
   }
}



cracklord.config(["$httpProvider", function($httpProvider) {
   $httpProvider.defaults.transformResponse.push(function(resData) {
      sortEnumInJSON(resData);
      return resData;
   });
}]);

cracklord.constant('USER_ROLES', {
	admin: 'Administrator',
	standard: 'Standard User',
	read: 'Read-Only'
});

cracklord.constant('JOB_STATUS_RUNNING', {
   running: 'running',
   paused: 'paused',
   created: 'created'
});

cracklord.constant('JOB_STATUS_COMPLETED', {
   done: 'done',
   failed: 'failed',
   quit: 'quit'
});

cracklord.constant('QUEUE_STATUS', {
   empty: 'empty',
   running: 'running',
   paused: 'paused',
   exhaused: 'exhausted'
});

cracklord.constant('RESOURCE_STATUS', {
   running: 'running',
   paused: 'paused'
});