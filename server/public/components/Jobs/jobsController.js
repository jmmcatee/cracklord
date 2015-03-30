cracklord.controller('JobsController', ['$scope', 'JobsService', 'growl', 'ResourceList', function JobsController($scope, JobsService, growl, ResourceList) {
	$scope.listreordered = false;
	$scope.now = Math.floor(Date.now() / 1000);
	$scope.jobactions = {};


	$scope.sortableOptions = {
		handle: '.draghandle',
		axis: 'y',
		update: function(e, ui) {
			$scope.listreordered = true;
		}
	};

	$scope.reorderConfirm = function() {

	}

	$scope.reorderCancel = function() {
		$scope.listreordered = false;
		$scope.loadJobs();
		growl.info("Job reorder was cancelled.")
	}

	$scope.loadJobs = function() {
		var jobs = JobsService.query(
			//Our success handler
			function(data) {
				$scope.jobs = data;

				$scope.listreordered = false;
				for(var i = 0; i < $scope.jobs.length; i++) {
					var resource = ResourceList.get($scope.jobs[i].resourceid);
					if(resource) {
						$scope.jobs[i].resourcecolor = "background-color: rgb("+resource.color.r+","+resource.color.g+","+resource.color.b+");";
					}
					$scope.jobs[i].expanded = false;
				}

			},
			//Our error handler
			function(error) {
				switch (error.status) {
					case 400: growl.error("You sent bad data, check your input and if it's correct get in touch with us on github"); break;
					case 403: growl.warning("You're not allowed to do that..."); break;
					case 404: growl.error("That object was not found."); break;
					case 409: growl.error("The request could not be completed because there was a conflict with the existing resource."); break;
					case 500: growl.error("An internal server error occured while trying to add the resource."); break;
				}
			}
		);
	}
	$scope.loadJobs();
}]);

cracklord.directive('jobDetail', ['JobsService', 'ResourceService', 'ToolsService', 'growl', 'ResourceList', function jobDetail(JobsService, ResourceService, ToolsService, growl, ResourceList) {
	return {
		restrict: 'E',
		templateUrl: 'components/Jobs/jobsViewDetail.html',
		scope: {
			jobid: '@',
			visibility: '='
		},
		controller: function($scope) {
			// Mmmmmmmm.... Donut.
			$scope.processDonut = function() {
				$scope.donut = {};
				$scope.donut.labels = ['Processed', 'Total'];

				var processed = $scope.detail.totalhashes * $scope.detail.progress;
				var total = $scope.detail.totalhashes - processed;
				$scope.donut.data = [processed, total];

				$scope.donut.colors = [ '#337ab7', '#aaaaaa' ];
			};

			$scope.processLine = function() {
				$scope.line = {};
				$scope.line.series = [ $scope.detail.performancetitle ]; 
				$scope.line.data = [];
				$scope.line.data[0] = [];
				$scope.line.labels = [];
				$scope.line.options = {
					'pointDot': false,
					'showTooltips': false
				};
				$scope.line.colors = [
					'#d43f3a'
				]

				for(var time in $scope.detail.performancedata) {
					$scope.line.data[0].push($scope.detail.performancedata[time]);
					$scope.line.labels.push("");
				}
			}
		},
		link: function($scope, $element, $attrs) {
			$scope.$watch('visibility', function(newval, oldval) {
				if(newval === true) {
					JobsService.get({id: $scope.jobid}, 
						function success(data) {
							ToolsService.get({id: data.job.toolid}, 
								function toolsuccess(data) {
									$scope.tool = data.tool;
								}
							);
							$scope.detail = data.job;

							var resource = ResourceList.get(data.job.resourceid);
							if(resource) {
								$scope.detail.resourcename = resource.name;
							}

							$scope.processDonut();
							$scope.processLine();

							$element.parent().show();
							$element.find('.slider').slideDown();
						},
						function error(error) {
							growl.error("There was a problem loading job details.")
							$($element).find('div.slider').slideUp("slow", function() {
								$element.parent().hide();
							});
						}
					);
				} else {
					$($element).find('div.slider').slideUp("slow", function() {
						$element.parent().hide();
					});
				}
			});
		},
	}
}]);

cracklord.filter('currentJobs', ['JOB_STATUS_RUNNING', 'JOB_STATUS_COMPLETED', function(JOB_STATUS_RUNNING, JOB_STATUS_COMPLETED) {
	return function(items) {
		var filtered = [];
		for (var i = 0; i < items.length; i++) {
			var item = angularjs.copy(items[i]);
			if(JOB_STATUS_RUNNING.indexOf(item.status)) {
				filtered.push(item);
			}
		}
		return filtered;
	};
}]);

cracklord.controller('CreateJobController', ['$scope', '$state', 'ToolsService', 'JobsService', 'growl', function CreateJobController($scope, $state, ToolsService, JobsService, growl) {
	$scope.formData = {};
	$scope.formData.params = {};

	$scope.toolChange = function() {
		var id = $scope.formData.tool.id;
		var tool = ToolsService.get({id: id}, 
			function(data) {
				$scope.tool = data.tool;
			}, 
			function(error) {
				growl.error("An error occured while trying to load tool information.");
			}
		);
	}

	$scope.processNewJobForm = function() {
		var newjob = new JobsService();

		newjob.toolid = $scope.formData.tool.id;
		newjob.name = $scope.formData.name;
		newjob.params = $scope.formData.params;
		
		JobsService.save(newjob, 
			function(data) {
				growl.success("Job successfully added");
				$state.transitionTo('jobs');
			}, 
			function(error) {
				switch (error.status) {
					case 400: growl.error("You sent bad data, check your input and if it's correct get in touch with us on github"); break;
					case 403: growl.warning("You're not allowed to do that..."); break;
					case 404: growl.error("That object was not found."); break;
					case 409: growl.error("The request could not be completed because there was a conflict with the existing resource."); break;
					case 500: growl.error("An internal server error occured while trying to add the resource."); break;
				}
			}
		);
	}	
}]);