cracklord.controller('ResourcesController', ['$scope', 'ResourceService', 'growl', function ResourcesController($scope, ResourceService, growl) {
	$scope.loadServers = function() {
		var servers = ResourceService.query(
			function(data) {
			}, 
			function(error) {
				switch (error.status) {
					case 400: growl.error("You sent bad data, check your input and if it's correct get in touch with us on github"); break;
					case 403: growl.error("You're not allowed to do that..."); break;
					case 409: growl.error("The request could not be completed because there was a conflict with the existing resource."); break;
					case 500: growl.error("An internal server error occured while trying to add the resource."); break;
				}
			}
		);
		$scope.resources = servers;
	}
	$scope.loadServers();
}]);

cracklord.controller('ConnectResourceController', ['$scope', '$state', 'ResourceService', 'growl', function CreateJobController($scope, $state, ResourceService, growl) {
	$scope.formData = {};

	$scope.processResourceConnectForm = function() {
		var newresource = new ResourceService();

		newresource.name = $scope.formData.name;
		newresource.address = $scope.formData.address;
		newresource.key = $scope.formData.key;
		
		ResourceService.save(newresource).$promise.then(
			function(data) {
				growl.success("Connecting to resource");
				$state.transitionTo('resources');
			}, 
			function(error) {
				switch (error.status) {
					case 400: growl.error("You sent bad data, check your input and if it's correct get in touch with us on github"); break;
					case 403: growl.error("You're not allowed to do that..."); break;
					case 409: growl.error("The request could not be completed because there was a conflict with the existing resource."); break;
					case 500: growl.error("An internal server error occured while trying to add the resource."); break;
				}
			}
		);
	}	
}]);