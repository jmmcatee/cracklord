var cracklord = angular.module('cracklord', [
	'ui.router',
   'ui.sortable',
   'ngResource',
   'schemaForm',
   'angular-growl',
   'readableTime',
   'ngAnimate',
   'ngCsv',
   'ngMockE2E',
   'chart.js'
]);

var interceptor = function($q, $location) {
   return {
      request: function(data) {
         console.log("%cHTTP REQUEST: %s\n%O\n%s", "color: blue;", data.url, data, JSON.stringify(data.data));
         return data;
      },
      response: function(result) {
         console.log("%cHTTP RESPONSE: %s\n%O\n%s", "color: green;", result.status, result, JSON.stringify(result.data));
         return result;
      },
      requestError: function(error) {
         console.log("%cHTTP REQUEST ERROR: %s\n%O", "color: orange;", error.url, error);
         return error;
      },
      responseError: function(error) {
         console.log("%cHTTP RESPONSE ERROR: %s\n%O", "color: red;", error.status, error);
         return error;
      }
   }
};

cracklord.config(['growlProvider', function ($growlProvider) {
  $growlProvider.globalTimeToLive(5000);
//  growlProvider.globalDisableCountDown(true);
}]);

cracklord.config(function ($httpProvider) {
   $httpProvider.interceptors.push(interceptor);
});

cracklord.constant('USER_ROLES', {
	admin: 'Administrator',
	standard: 'Standard User',
	read: 'Read-Only'
});