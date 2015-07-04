cracklord.service('UserSession', ['$cookies', function($cookies) {
	this.create  = function(userToken, userName, userRole) {
		this.token = userToken
		$cookies.put('usertoken', userToken);
		this.name = userName;
		$cookies.put('username', userName);
		this.role = userRole;
		$cookies.put('userrole', userRole);
	}
	this.destroy = function() {
		this.token = null
		$cookies.remove('usertoken');
		this.name = null;
		$cookies.remove('username');
		this.role = null;
		$cookies.remove('userrole');
	}
	this.initCookies = function() {
		this.token = $cookies.get('usertoken');
		this.name = $cookies.get('username');
		this.role = $cookies.get('userrole');
		if(this.token && this.name && this.role) {
			return true;
		} 
		return false;
	}
	return this;
}]);

cracklord.factory('AuthService', ['$http', 'UserSession', function($http, UserSession) {
	var authService = {};
	UserSession.initCookies();

	authService.login = function(creds) {
		return $http.post('/api/login', creds);
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

	authService.logout = function() {
		return $http.get('/api/logout');
	};

	return authService;
}]);

cracklord.factory('userTokenHttpInterceptor', ['$q', 'UserSession', '$injector', 'growl', function($q, UserSession, $injector, growl) {
	return {
		request: function(req) {
			if(req.url.startsWith('/api')) {
				if(req.url !== '/api/login') {
					req.headers = req.headers || {};
					if(UserSession.token) {
						req.headers.AuthorizationToken = UserSession.token;
					}
				}	
			}
			return req;
		},
		responseError: function(res) {
			if(res.status === 401) {
				$injector.get('$state').go('login');
				if(req.url !== '/api/login') {
					growl.warning("You need to login first.");
				}
			}
			return $q.reject(res);
		}
	}
}]);

cracklord.config(['$httpProvider', function($httpProvider) {
	$httpProvider.interceptors.push('userTokenHttpInterceptor');
}]);

cracklord.run(['$rootScope', '$state', 'AuthService', 'growl', function($rootScope, $state, AuthService, growl) {
	$rootScope.$on('$stateChangeStart', function(event, next) {
		if(next.data.authorizedRoles.length) {
			if(!AuthService.isAuthorized(next.data.authorizedRoles)) {
				event.preventDefault();
				if(AuthService.isAuthenticated()) {
					growl.warning("Ah ah ah! You didn't say the magic word!");
				} else {
					$state.go('login');
				}
			}
		}
	})
}]);