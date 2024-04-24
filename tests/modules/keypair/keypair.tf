resource "edgecenter_keypair" "kp" {
  project_id = var.project_id
  public_key  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBpNg06SMZ5B2f4pkjHcErJbW04pTiSyEGSrvabalI6T terratest_keypair"
  sshkey_name = var.keypair_name
}