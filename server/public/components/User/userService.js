cracklord.service('UserSession', function() {
	this.create  = function(userToken, userName, userRole) {
		this.token = userToken;
		this.name = userName;
		this.role = userRole;
	}
	this.destroy = function() {
		this.token = null;
		this.name = null;
		this.role = null;
	}
	return this;
});

cracklord.factory('AuthService', function($http, UserSession) {
	var authService = {};

	authService.login = function(creds) {
		return $http
			.post('/api/login', creds)
			.then(function (result) {
				UserSession.create(res.data.token, res.data.username, res.data.userrole);
			});
	};

	authService.isAuthenticated = function() {
		return !!UserSession.name;
	};

	authService.isAuthorized = function(allowedRoles) {
		if(!angular.isArray(allowedRoles)) {
			allowedRoles = [allowedRoles];
		}

		return (authService.isAuthenticated() && allowedRoles.indexOf(UserSession.role) !== -1);
	};

	return authService;
});

cracklord.run(function($rootScope, $state, AuthService, growl) {
	$rootScope.$on('$stateChangeStart', function(event, next) {
		if(next.data.authorizedRoles.length) {
			if(!AuthService.isAuthorized(next.data.authorizedRoles)) {
				event.preventDefault();
				if(AuthService.isAuthenticated()) {
					growl.warning("You're not allowed to do that.");
				} else {
					$state.go('login');
				}
			}
		}
	})
})