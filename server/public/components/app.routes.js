cracklord.config(['$stateProvider', '$urlRouterProvider', 'USER_ROLES', function($stateProvider, $urlRouterProvider, USER_ROLES) {
	$urlRouterProvider.otherwise('/jobs');
	$stateProvider
		.state('jobs', {
			url:'/jobs',
			templateUrl: 'components/Jobs/jobsView.html',
			data: {
				authorizedRoles: [USER_ROLES.admin, USER_ROLES.standard, USER_ROLES.read]
			}
		})
		.state('resources', {
			url:'/resources',
			templateUrl: 'components/Resources/resourcesView.html',
			data: {
				authorizedRoles: [USER_ROLES.admin]
			}
		})
		.state('connectresource', {
			url:'/resources/new',
			templateUrl: 'components/Resources/connectResource.html',
			data: {
				authorizedRoles: [USER_ROLES.admin]
			}
		})
		.state('createjob', {
			url:'/jobs/new',
			templateUrl: 'components/Jobs/newJob.html',
			controller: 'CreateJobController',
			data: {
				authorizedRoles: [USER_ROLES.admin, USER_ROLES.standard]
			}
		})
		.state('createjob.tools', {
			url:"/tools",
			templateUrl: 'components/Jobs/newJob.tools.html',
			data: {
				authorizedRoles: [USER_ROLES.admin, USER_ROLES.standard]
			}
		})
		.state('createjob.details', {
			url:"/details",
			templateUrl: 'components/Jobs/newJob.details.html',
			data: {
				authorizedRoles: [USER_ROLES.admin, USER_ROLES.standard]
			}
		})
		.state('login', {
			url:'/login', 
			templateUrl: 'components/User/login.html',
			data: {
				authorizedRoles: []
			}
		});
}]);