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

cracklord.config(['growlProvider', function (growlProvider) {
  growlProvider.globalTimeToLive(5000);
//  growlProvider.globalDisableCountDown(true);
}]);

cracklord.constant('USER_ROLES', {
	admin: 'Administrator',
	standard: 'Standard User',
	read: 'Read-Only'
});