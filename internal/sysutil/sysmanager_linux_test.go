package sysutil

import (
	"errors"
	mock_os "github.com/digitalocean/droplet-agent/internal/sysutil/internal/mocks"
	"go.uber.org/mock/gomock"
	"io"
	"os"
	"reflect"
	"syscall"
	"testing"
)

func TestSysManager_ReadFileOfUser(t *testing.T) {
	type args struct {
		filename string
		user     *User
	}
	tests := []struct {
		name    string
		prepare func(op *MockosOperator, f *MockFile, fi *mock_os.MockFileInfo)
		args    args
		want    []byte
		wantErr error
	}{
		{
			name: "return ErrOpenFileFailed if failed to open file",
			prepare: func(op *MockosOperator, f *MockFile, fi *mock_os.MockFileInfo) {
				op.EXPECT().openFile("/var/log/file", os.O_RDONLY, os.FileMode(0)).Return(nil, errors.New("open-err"))
			},
			args: args{
				filename: "/var/log/file",
				user:     &User{UID: 123},
			},
			want:    nil,
			wantErr: ErrOpenFileFailed,
		},
		{
			name: "return ErrUnexpected if failed to stat file",
			prepare: func(op *MockosOperator, f *MockFile, fi *mock_os.MockFileInfo) {
				op.EXPECT().openFile("/var/log/file", os.O_RDONLY, os.FileMode(0)).Return(f, nil)
				f.EXPECT().Stat().Return(nil, errors.New("stat-err"))
				f.EXPECT().Close()
			},
			args: args{
				filename: "/var/log/file",
				user:     &User{UID: 123},
			},
			want:    nil,
			wantErr: ErrUnexpected,
		},
		{
			name: "return ErrInvalidFileType if file not regular",
			prepare: func(op *MockosOperator, f *MockFile, fi *mock_os.MockFileInfo) {
				op.EXPECT().openFile("/var/log/file", os.O_RDONLY, os.FileMode(0)).Return(f, nil)
				f.EXPECT().Stat().Return(fi, nil)
				fi.EXPECT().Mode().Return(os.ModeType)
				f.EXPECT().Close()
			},
			args: args{
				filename: "/var/log/file",
				user:     &User{UID: 123},
			},
			want:    nil,
			wantErr: ErrInvalidFileType,
		},
		{
			name: "non-root return ErrUnexpected if failed to convert to sys stat",
			prepare: func(op *MockosOperator, f *MockFile, fi *mock_os.MockFileInfo) {
				op.EXPECT().openFile("/var/log/file", os.O_RDONLY, os.FileMode(0)).Return(f, nil)
				f.EXPECT().Stat().Return(fi, nil)
				fi.EXPECT().Mode().Return(os.ModeTemporary)
				fi.EXPECT().Sys().Return(&struct{}{})
				f.EXPECT().Close()
			},
			args: args{
				filename: "/var/log/file",
				user:     &User{UID: 123},
			},
			want:    nil,
			wantErr: ErrUnexpected,
		},
		{
			name: "non-root return ErrPermissionDenied if ownership not match",
			prepare: func(op *MockosOperator, f *MockFile, fi *mock_os.MockFileInfo) {
				op.EXPECT().openFile("/var/log/file", os.O_RDONLY, os.FileMode(0)).Return(f, nil)
				f.EXPECT().Stat().Return(fi, nil)
				fi.EXPECT().Mode().Return(os.ModeTemporary)
				fi.EXPECT().Sys().Return(&syscall.Stat_t{Uid: 321})
				f.EXPECT().Close()
			},
			args: args{
				filename: "/var/log/file",
				user:     &User{UID: 123},
			},
			want:    nil,
			wantErr: ErrPermissionDenied,
		},
		{
			name: "non-root happy path",
			prepare: func(op *MockosOperator, f *MockFile, fi *mock_os.MockFileInfo) {
				op.EXPECT().openFile("/var/log/file", os.O_RDONLY, os.FileMode(0)).Return(f, nil)
				f.EXPECT().Stat().Return(fi, nil)
				fi.EXPECT().Mode().Return(os.ModeTemporary)
				fi.EXPECT().Sys().Return(&syscall.Stat_t{Uid: 123})
				f.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
				f.EXPECT().Close()
			},
			args: args{
				filename: "/var/log/file",
				user:     &User{UID: 123},
			},
			want:    []byte{},
			wantErr: nil,
		},
		{
			name: "root happy path",
			prepare: func(op *MockosOperator, f *MockFile, fi *mock_os.MockFileInfo) {
				op.EXPECT().openFile("/var/log/file", os.O_RDONLY, os.FileMode(0)).Return(f, nil)
				f.EXPECT().Stat().Return(fi, nil)
				fi.EXPECT().Mode().Return(os.ModeTemporary)
				f.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
				f.EXPECT().Close()
			},
			args: args{
				filename: "/var/log/file",
				user:     &User{UID: 0},
			},
			want:    []byte{},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			osOpMock := NewMockosOperator(mockCtl)
			fileMock := NewMockFile(mockCtl)
			fileInfoMock := mock_os.NewMockFileInfo(mockCtl)
			if tt.prepare != nil {
				tt.prepare(osOpMock, fileMock, fileInfoMock)
			}
			s := &SysManager{
				osOperator: osOpMock,
			}
			got, err := s.ReadFileOfUser(tt.args.filename, tt.args.user)
			if (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("ReadFileOfUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadFileOfUser() got = %v, want %v", got, tt.want)
			}
		})
	}
}
