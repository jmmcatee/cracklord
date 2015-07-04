cracklord.controller('UserController', ['$state', '$scope', 'USER_ROLES', 'AuthService', 'UserSession', function UserController($state, $scope, USER_ROLES, AuthService, UserSession) {
	$scope.user = {}
	$scope.user.name = UserSession.name;
	$scope.user.allroles = USER_ROLES;
	$scope.user.checkrole = AuthService.isAuthorized;

	$scope.userLogout = function() {
		AuthService.logout()
			.success(function(data, status, headers, config) {
					$scope.name = null;
					UserSession.destroy();
					$state.go('login');
			})
			.error(function (data, status, headers, config) {
				growl.error('An error occured while trying to log you out.');
			});
	}
}]);

cracklord.controller('LoginFormController', ['$state', '$scope', 'AuthService', 'growl', 'UserSession', function LoginFormController($state, $scope, AuthService, growl, UserSession) {
	$scope.login = {};
	$scope.login.failed = false;

	$scope.processLoginForm = function() {
		var creds = {
			username: $scope.login.username, 
			password: $scope.login.password
		}
		AuthService.login(creds)
			.success(function(data, status, headers, config) {
				UserSession.create(data.token, $scope.login.username, data.role);
				$scope.user.name = UserSession.name;
				growl.success("Login successful.");
				$state.go('jobs');
			})
			.error(function (data, status, headers, config) {
				$scope.login.failed = true;
				growl.error("Login failed.");
			});
	};
}]);

