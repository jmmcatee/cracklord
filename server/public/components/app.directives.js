cracklord.directive("confirmPopover", function() {
    return {
        restrict: 'A',
        link: function (scope, el, attrs) {
            var id = scope.target.id;
            var html = '<div id="confirm_'+id+'" class="btn-group"> <button type="button" class="btn btn-small btn-success"> <i class="fa fa-check-circle"></i> Yes</button><button type="button" class="btn btn-small btn-danger"><i class="fa fa-times-circle"></i> No</button></div>';

            $(el).popover({
                trigger: 'manual',
                html: true,
                title: attrs.popoverText || "Are you sure?",
                content: html,
                placement: attrs.popoverPlacement || "top",
                container: 'body'
            });

            el.bind('click', function(e) {
                e.stopPropagation();
                el.popover('show');
                var popoverDiv = $("#confirm_"+id)
                popoverDiv.find(".btn-success").click(function(e) {
                    el.popover('hide');
                    scope.$apply(attrs.confirmPopover);
                });
                popoverDiv.find(".btn-danger").click(function(e) {
                    el.popover('hide');
                });
            });

        }
    };
});

cracklord.directive('playbutton', function splaybutton(growl, AuthService, USER_ROLES) {
    return {
        restrict: 'E',
        replace: true,
        template: '<button aria-label="play" type="button" class="btn btn-success"><i class="fa fa-fw fa-play"></i></button>',
        scope: {
            target: '='         
        },
        controller: function($scope) {
            $scope.doOnClick = function() {
                if($scope.target.status === 'created' || $scope.target.status === 'running') {
                    $scope.target.status='paused';
                } else if ($scope.target.status === 'paused') {
                    $scope.target.status = 'created';
                }
                $scope.target.$update({id: $scope.target.id}, 
                    function success(successResult) {
                        if($scope.target.status === 'created' || $scope.target.status === 'running') {
                            growl.success($scope.target.name+' resumed successfully.');
                        } else if ($scope.target.status === 'paused') {
                            growl.success($scope.target.name+' paused successfully.')
                        }
                    }, 
                    function error(errorResult) {
                        growl.error("Error "+errorResult.message);
                    }
                );
            }
        },
        link: function($scope, $element, $attrs) {
            if(!AuthService.isAuthorized([USER_ROLES.standard, USER_ROLES.admin])) {
                $scope.$destroy();
                $element.remove();
            } else {
                $scope.$watch('target.status', function (newval, oldval) {
                    if(newval === 'created' || newval === 'running') {
                        $element.removeClass('btn-success');
                        $element.children('i').removeClass('fa-play');
                        $element.addClass('btn-warning');
                        $element.children('i').addClass('fa-pause');
                    } else if (newval === 'paused') {
                        $element.removeClass('btn-warning');
                        $element.children('i').removeClass('fa-pause');
                        $element.addClass('btn-success');
                        $element.children('i').addClass('fa-play');
                    }
                });
                $element.bind('click', function() {
                    $scope.doOnClick();
                });
            }
        }
    };
});

cracklord.directive('stopbutton', function stopbutton(growl, AuthService, USER_ROLES) {
    return {
        restrict: 'E', 
        replace: true,
        template: '<button confirm-popover="doClickConfirm()" aria-label="stop" type="button" class="btn btn-danger"><i class="fa fa-fw fa-stop"></i></button>',
        scope: {
            target: '='
        },
        controller: function($scope) {
            $scope.doClickConfirm = function() {
                $scope.target.status = 'quit';
                $scope.target.$update({id: $scope.target.id}, 
                    function success(successResult) {
                        growl.success($scope.target.name+' stopped.');
                    },
                    function error(errorResult) {
                        growl.error("Error "+errorResult.message);
                    }
                );
            }   
        },
        link: function($scope, $element, $attrs) {
            if(!AuthService.isAuthorized([USER_ROLES.standard, USER_ROLES.admin])) {
                $scope.$destroy();
                $element.remove();
            }
        }
    };
});

cracklord.directive('trashbutton', function trashbutton(growl, AuthService, USER_ROLES) {
    return {
        restrict: 'E',
        replace: true,
        template: '<button confirm-popover="doClickConfirm()" aria-label="delete" type="button" class="btn btn-danger"><i class="fa fa-fw fa-trash-o"></i></button>',
        scope: {
            target: '=',
            targetlist: '='
        },
        controller: function($scope) {
            $scope.doClickConfirm = function() {
                var index = $scope.targetlist.map(function(el) {
                    return el.id;
                }).indexOf($scope.target.id);

                var name = $scope.target.name;
                $scope.target.$delete({id: $scope.target.id}, 
                    function success(successResult) {
                        growl.success(name+" was deleted.");
                        $scope.targetlist.splice(index, 1);
                    }, 
                    function error(errorResult) {
                        growl.error("Error "+errorResult.message);
                    }
                );
            }
        },
        link: function($scope, $element, $attrs) {
            if(!AuthService.isAuthorized([USER_ROLES.standard, USER_ROLES.admin])) {
                $scope.$destroy();
                $element.remove();
            }   
        }
    };
});

cracklord.directive('draghandle', function draghandle(growl, AuthService, USER_ROLES) {
    return {
        restrict: 'E',
        replace: true,
        template: '<div class="btn btn-primary draghandle"><i class="fa fa-fw fa-arrows-v"></i></div>',
        link: function($scope, $element, $attrs) {
            if(!AuthService.isAuthorized([USER_ROLES.standard, USER_ROLES.admin])) {
                $element.remove();
            }   
        }
    };
});


cracklord.directive('rcSubmit', ['$parse', function ($parse) {
    return {
        restrict: 'A',
        require: ['rcSubmit', '?form'],
        controller: ['$scope', function ($scope) {
            this.attempted = false;
            
            var formController = null;
            
            this.setAttempted = function() {
                this.attempted = true;
            };
            
            this.setFormController = function(controller) {
              formController = controller;
            };
            
            this.needsAttention = function (fieldModelController) {
                if (!formController) return false;
                
                if (fieldModelController) {
                    return fieldModelController.$invalid && (fieldModelController.$dirty || this.attempted);
                } else {
                    return formController && formController.$invalid && (formController.$dirty || this.attempted);
                }
            };
        }],
        compile: function(cElement, cAttributes, transclude) {
            return {
                pre: function(scope, formElement, attributes, controllers) {
                  
                    var submitController = controllers[0];
                    var formController = (controllers.length > 1) ? controllers[1] : null;
                    
                    submitController.setFormController(formController);
                    
                    scope.rc = scope.rc || {};
                    scope.rc[attributes.name] = submitController;
                },
                post: function(scope, formElement, attributes, controllers) {
                  
                    var submitController = controllers[0];
                    var formController = (controllers.length > 1) ? controllers[1] : null;
                    var fn = $parse(attributes.rcSubmit);
                    
                    formElement.bind('submit', function (event) {
                        submitController.setAttempted();
                        if (!scope.$$phase) scope.$apply();
                        
                        if (!formController.$valid) return false;
                
                        scope.$apply(function() {
                            fn(scope, {$event:event});
                        });
                    });
                }
          };
        }
    };
}]);