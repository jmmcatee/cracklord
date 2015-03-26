var cracklord = angular.module('cracklord', [
	'ui.router',
   'ui.sortable',
   'ngResource',
   'schemaForm',
   'angular-growl',
   'readableTime',
   'ngAnimate',
   'ngCsv',
   'chart.js'
]);

cracklord.config(['growlProvider', function ($growlProvider) {
  $growlProvider.globalTimeToLive(5000);
//  growlProvider.globalDisableCountDown(true);
}]);


var regexISO8601 = /^(\d{4}|\+\d{6})(?:-(\d{2})(?:-(\d{2})(?:T(\d{2}):(\d{2}):(\d{2})\.(\d{1,})(Z|([\-+])(\d{2}):(\d{2}))?)?)?)?$/;
function convertDateStringsToDates(input) {
   if (typeof input !== "object") return input;

   for (var key in input) {
      if (!input.hasOwnProperty(key)) continue;

      var value = input[key];
      var match;
      if (typeof value === "string" && (match = value.match(regexISO8601))) {
         var milliseconds = Date.parse(match[0]);
         if (!isNaN(milliseconds)) {
            input[key] = new Date(milliseconds);
         }
      } else if (typeof value === "object") {
         convertDateStringsToDates(value);
      }
   }
}
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
      convertDateStringsToDates(resData);
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