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

cracklord.controller('ConnectResourceController', function CreateJobController($scope, $state, ResourceService, growl) {
	$scope.formData = {};

	$scope.processResourceConnectForm = function() {
		var newresource = new ResourceService();

		newresource.name = $scope.formData.name;
		newresource.address = $scope.formData.address;
		newresource.key = $scope.formData.key;
		
		ResourceService.save(newresource, 
			function(data) {
				growl.success("Connecting to resource");
				$state.transitionTo('resources');
			}, 
			function(error) {
				growl.error("An error occured while trying to connect to the resource.");
			}
		);
	}	
});