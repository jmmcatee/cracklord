cracklord.service('UserSession', function() {
	this.create  = function(userToken, userName, userRole) {
		this.token = userToken
		sessionStorage.setItem('usertoken', userToken);
		this.name = userName;
		sessionStorage.setItem('username', userName);
		this.role = userRole;
		sessionStorage.setItem('userrole', userRole);
	}
	this.destroy = function() {
		this.token = null
		sessionStorage.removeItem('usertoken');
		this.name = null;
		sessionStorage.removeItem('username');
		this.role = null;
		sessionStorage.removeItem('userrole');
	}
	this.initCookies = function() {
		this.token = sessionStorage.getItem('usertoken');
		this.name = sessionStorage.getItem('username');
		this.role = sessionStorage.getItem('userrole');
		return this.token;
	}
	return this;
});

cracklord.factory('AuthService', function($http, UserSession) {
	var authService = {};

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
		return $http.get('/api/logout?token='+UserSession.getToken());
	};

	return authService;
});

cracklord.factory('userTokenHttpInterceptor', function($q, UserSession) {
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
		}
	}
});

cracklord.config(function($httpProvider) {
	$httpProvider.interceptors.push('userTokenHttpInterceptor');
});

cracklord.run(function($rootScope, $state, AuthService, growl) {
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
});