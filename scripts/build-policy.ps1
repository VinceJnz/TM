function Get-BuildRevision([string]$Revision,[bool]$Dirty,[bool]$RequireClean=$false){
 if($Revision -cnotmatch '^[a-f0-9]{40}$'){throw 'Invalid Git revision.'}
 if($Dirty -and $RequireClean){throw 'Release builds require a clean working tree.'}
 if($Dirty){return "$Revision-dirty"}
 return $Revision
}
