cracklord.controller('ResourcesController', function ResourcesController($scope, ResourceService, growl) {
	$scope.loadServers = function() {
		var servers = ResourceService.query(
			function(data) {
				growl.success("Resources successfully loaded.");
			}, 
			function(error) {
				growl.error("There was an error loading resources.");
			}
		);
		$scope.resources = servers;
	}
	$scope.loadServers();
});