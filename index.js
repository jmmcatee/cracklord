var index = angular.module('index', []);
index.controller('PluginController', ['$scope', '$http', function ($scope, $http) {
	$scope.plugins = {}
	$http.get('plugins/tools.json').success(function(data) {
		$scope.plugins.tools = data;
	});
	$http.get('plugins/resourcemanagers.json').success(function(data) {
		$scope.plugins.resourcemanagers = data;
	});
}]);
index.filter('unsafe', ['$sce', function($sce) { return $sce.trustAsHtml; }]);

$('body').scrollspy({ target: '#navbar' })
$('.carousel').carousel({
	interval: 5000,
	pause: "false"
})