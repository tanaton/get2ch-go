#get2ch-go
##�C���X�g�[��
`
$ go get github.com/tanaton/get2ch-go
`
##�ȒP�Ȏg����
````Go
// �g�p�O�̏����i���Ăׂ�OK�j
get2ch.Start(get2ch.NewFileCache("/2ch/dat"), nil)
// �擾�������X���b�h���w�肵��
get := get2ch.NewGet2ch("huga", "1234567890")
// �f�[�^���擾
data, err := get.GetData()
````
��̂̎g������example/readd.go������Ε����邩��
