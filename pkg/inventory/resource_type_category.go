package inventory

var categoryMap = map[string][]string{
	"Identity & Access": []string{
		"aws::iam::user",
		"aws::iam::group",
		"aws::iam::policy",
		"aws::iam::policyattachment",
		"aws::iam::role",
		"aws::iam::accessadvisor",
		"aws::iam::accountpasswordpolicy",
		"aws::identitystore::groupmembership",
		"aws::identitystore::user",
		"aws::identitystore::group",
		"aws::ssoadmin::accountassignment",
		"aws::ssoadmin::permissionset",
		"aws::ssoadmin::attachedmanagedpolicy",
		"aws::ssoadmin::instance",
		"microsoft.authorization/roleassignment",
		"microsoft.authorization/policyassignments",
		"microsoft.authorization/roledefinitions",
	},
	"Entra ID Directory": []string{
		"microsoft.entra/users",
		"microsoft.entra/directoryauditreport",
		"microsoft.entra/userregistrationdetails",
		"microsoft.entra/groups",
	},
}
