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
cracklord.config(["$httpProvider", function($httpProvider) {
   $httpProvider.defaults.transformResponse.push(function(resData) {
      convertDateStringsToDates(resData);
      return resData;
   });
}]);

cracklord.constant('USER_ROLES', {
	admin: 'Administrator',
	standard: 'Standard User',
	read: 'Read-Only'
});