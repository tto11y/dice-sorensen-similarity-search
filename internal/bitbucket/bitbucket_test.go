package bitbucket_test

import (
	"dice-sorensen-similarity-search/internal/bitbucket"
	"dice-sorensen-similarity-search/internal/config"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/logging"
	"errors"
	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"testing"
)

type MockBitbucketAdapter struct {
	StreamFilesResponse   *bitbucketv1.APIResponse
	GetRawContentResponse *bitbucketv1.APIResponse
	GetContentResponse    *bitbucketv1.APIResponse
	Error                 error
}

func (m *MockBitbucketAdapter) GetContent(projectKey, repositorySlug string, localVarOptionals map[string]any) (*bitbucketv1.APIResponse, error) {
	return m.GetContentResponse, m.Error
}

func (m *MockBitbucketAdapter) GetRawContent(projectKey, repositorySlug, path string, localVarOptionals map[string]any) (*bitbucketv1.APIResponse, error) {
	return m.GetRawContentResponse, m.Error
}

func (m *MockBitbucketAdapter) StreamFiles(projectKey, repositorySlug string, localVarOptionals map[string]any) (*bitbucketv1.APIResponse, error) {
	return m.StreamFilesResponse, m.Error
}

func TestReadRepoRootFolderContent(t *testing.T) {
	tests := []struct {
		name           string
		adapter        bitbucket.BitbucketApiServiceAdapter
		expectError    bool
		expectedResult []string
	}{
		{
			name:           "nilAdapter",
			adapter:        nil,
			expectError:    true,
			expectedResult: nil,
		},
		{
			name:           "nilResponse",
			adapter:        &MockBitbucketAdapter{},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name:           "noValuesContent",
			adapter:        &MockBitbucketAdapter{GetContentResponse: &bitbucketv1.APIResponse{}},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name:           "getContentReturnsError",
			adapter:        &MockBitbucketAdapter{GetContentResponse: nil, Error: errors.New("getContent")},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name: "childrenIsMissing",
			adapter: &MockBitbucketAdapter{GetContentResponse: &bitbucketv1.APIResponse{
				Values: map[string]any{},
			}},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name: "childrenIsNil",
			adapter: &MockBitbucketAdapter{GetContentResponse: &bitbucketv1.APIResponse{
				Values: map[string]any{
					"children": nil,
				},
			}},
			expectError:    false,
			expectedResult: []string{},
		},
		{
			name: "childrenIsNotMap",
			adapter: &MockBitbucketAdapter{GetContentResponse: &bitbucketv1.APIResponse{
				Values: map[string]any{
					"children": []string{},
				},
			}},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name: "valuesIsNotAnySlice",
			adapter: &MockBitbucketAdapter{GetContentResponse: &bitbucketv1.APIResponse{
				Values: map[string]any{
					"children": map[string]any{
						"values": 1,
					},
				},
			}},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name: "childOfValuesIsNotMap",
			adapter: &MockBitbucketAdapter{GetContentResponse: &bitbucketv1.APIResponse{
				Values: map[string]any{
					"children": map[string]any{
						"values": []any{
							1,
						},
					},
				},
			}},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name: "pathIsNotMap",
			adapter: &MockBitbucketAdapter{GetContentResponse: &bitbucketv1.APIResponse{
				Values: map[string]any{
					"children": map[string]any{
						"values": []any{
							map[string]any{
								"path": 34,
							},
						},
					},
				},
			}},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name: "nameIsNotString",
			adapter: &MockBitbucketAdapter{GetContentResponse: &bitbucketv1.APIResponse{
				Values: map[string]any{
					"children": map[string]any{
						"values": []any{
							map[string]any{
								"path": map[string]any{
									"name": 34,
								},
							},
						},
					},
				},
			}},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name: "childrenIsValid",
			adapter: &MockBitbucketAdapter{GetContentResponse: &bitbucketv1.APIResponse{
				Values: map[string]any{
					"children": map[string]any{
						"values": []any{
							map[string]any{
								"path": map[string]any{"name": "file1.txt"},
							},
							map[string]any{
								"path": map[string]any{"name": "file2.txt"},
							},
							map[string]any{
								"path": map[string]any{"name": ".gitignore"},
							},
							map[string]any{
								"path": map[string]any{"name": "markdowns"},
							},
							map[string]any{
								"path": map[string]any{"name": "images"},
							},
							map[string]any{
								"path": map[string]any{"name": "another-folder"},
							},
							map[string]any{
								"path": map[string]any{"name": "my-folder"},
							},
						},
					},
				},
			}},
			expectError:    false,
			expectedResult: []string{"file1.txt", "file2.txt", ".gitignore", "markdowns", "images", "another-folder", "my-folder"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &bitbucket.O11yBitbucketReader{Adapter: tt.adapter}

			gotRootFolderContent, err := reader.ReadRepoRootFolderContent("test_project", "test_repo")

			if tt.expectError {
				if err == nil {
					t.Fatalf("want error, but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("want NO error, but got: %v", err)
				}
			}

			if !cmp.Equal(gotRootFolderContent, tt.expectedResult) {
				t.Error(cmp.Diff(tt.expectedResult, gotRootFolderContent))
			}
		})
	}
}

func TestReadFileContentAtRevision(t *testing.T) {
	tests := []struct {
		name           string
		adapter        bitbucket.BitbucketApiServiceAdapter
		expectError    bool
		expectedResult string
	}{
		{
			name:           "nilAdapter",
			adapter:        nil,
			expectError:    true,
			expectedResult: "",
		},
		{
			name:           "nilResponse",
			adapter:        &MockBitbucketAdapter{},
			expectError:    true,
			expectedResult: "",
		},
		{
			name:           "noPayloadContent",
			adapter:        &MockBitbucketAdapter{GetRawContentResponse: &bitbucketv1.APIResponse{}},
			expectError:    false,
			expectedResult: "",
		},
		{
			name:           "getRawContentReturnsError",
			adapter:        &MockBitbucketAdapter{GetRawContentResponse: nil, Error: errors.New("getRawContent")},
			expectError:    true,
			expectedResult: "",
		},
		{
			name:           "PayloadIsValid",
			adapter:        &MockBitbucketAdapter{GetRawContentResponse: &bitbucketv1.APIResponse{Payload: []byte(getDummyFileContent())}},
			expectError:    false,
			expectedResult: getDummyFileContent(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &bitbucket.O11yBitbucketReader{Adapter: tt.adapter}

			gotFileContentAtRevision, err := reader.ReadFileContentAtRevision("test_project", "test_repo", "my-path", "1")

			if tt.expectError {
				if err == nil {
					t.Fatalf("want error, but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("want NO error, but got: %v", err)
				}
			}

			if !cmp.Equal(gotFileContentAtRevision, tt.expectedResult) {
				t.Error(cmp.Diff(tt.expectedResult, gotFileContentAtRevision))
			}
		})
	}
}

func getDummyFileContent() string {
	return "# Step 1: Gather Your Information\n\nFor the onboarding process there are several Organisational information that is needed which will be uploaded later in Service Now. \n\n| Information needed                                   | Example                                                                                                                              |\n| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |\n| Solution Code                                        | e.g.: 2DA31                                                                                                                          |\n| Solution Name                                        | e.g.: CLOUDSTREAM                                                                                                                    |\n| **three contacts for your solution**                 | **e.g.: (Solution manager, TAM, ...)**                                                                                               |\n| Contact Name                                         | e.g.: Hans Zimmermann                                                                                                                |\n| Contact Name                                         | e.g.: ...                                                                                                                            |\n| Contact Name                                         | e.g.: ...                                                                                                                            |\n|                                                      |                                                                                                                                      |\n|                                                      |                                                                                                                                      |\n| **Hostlist of the Systems that you want to monitor** |                                                                                                                                      |\n|                                                      |                                                                                                                                      |\n| Source Environment                                   | e.g.:<br>Production Data<br><br>- PROD<br>- QSYS<br><br>Non-Production <br><br>- Data<br>- DEV<br>- FAT<br>- UAT<br>- ABNA<br>- MONA |\n| IP                                                   | e.g.:                                                                                                                                |\n| Fully qualified domain Name                          | e.g.:                                                                                                                                |\n| System                                               | e.g.:RHEL, Windows, F5, Haproxy, Lamp,etc...                                                                                         |\n|                                                      |                                                                                                                                      |\n\n## Step 2 - Upload gathered information\n\nOnce all required information has been gathered, the formal request can be uploaded in [Service Now](https://servus.service-now.com/sp?id=im_cat_item&sys_id=8b6a088c87c47994028f631c8bbb358a) \n\n\n![](https://stash.s-mxs.net/projects/CIM/repos/o11y-self-service-content/raw/images/gatewayImg1ObservabilityGrafanaRequest.png)\n\nUpon receival, the data will be processed and the required Firewall-Clearance will be set up from our side.\nWhile waiting for the acceptance you can complete the procedure with step 3\n\n## Step 3 - ISD Update\n\nIn this step, you will update your ISD from your side. Align your update information with the table below. \n\n>[!NOTE]\n>- You as a data owner will allow us to monitor you with this DF. \n>- Also, security will be aware what kind of data you are sending to us\n>- This has nothing to do with technical implementation - that is covered in the Monitoring ISD! -> You do not need to add anything else or change architecture diagrams or so.\n\n>[!NOTE] \n> Next Step\n> Once processed, you will be contacted with an update or if needed we will align with you for a kickoff or followup meeting.\n  \n\n| **Dataflow Description ID** | **Dataflow ID** | **Source Solution**        | **Source Name**       | **Source Tenant**       | **Source Environment**                                                                                                                                    | **Source Zone**       | **Destination Solution**          | **Destination Name**    | **Destination Tenant** | **Destination Environment**                                       | **Destination Zone** | **Protocol** | **Ports** | **Transport Protection**                          | **Payload Protection**                            | **Description**                                                                                                                                                                                                                                                                                                                                        | **Justification**                                                                                                                                                     | **Cross-Environment Connections** | **Obsolete** | **Last Updated** |\n| --------------------------- | --------------- | -------------------------- | --------------------- | ----------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------- | --------------------------------- | ----------------------- | ---------------------- | ----------------------------------------------------------------- | -------------------- | ------------ | --------- | ------------------------------------------------- | ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------- | ------------ | ---------------- |\n| DF-XXXXXXXX                 |                 | ***Insert your Solution*** | ***Please fill out*** | ***ED or EG or EB AT*** | ***Production Data<br><br>- PROD<br><br>- QSYS<br><br>Non-Production <br><br>- Data<br><br>- DEV<br><br>- FAT<br><br>- UAT<br><br>- ABNA<br><br>- MONA*** | ***Please fill out*** | AR341 - Infrastructure Monitoring | Zabbix Monitoring Proxy | ED                     | Production Data<br><br>- PROD<br><br>Non-Production <br><br>- UAT | Application          |              |           | ***In case of business data -> please fill out*** | ***In case of business data -> please fill out*** | ***DF for Data-Owners defining Data that is sent. <br><br>  <br><br>Data that is sent to Monitoring: <br><br>- Infrastructure & Performance Data<br><br>- Business Data<br><br>  <br><br>Business Data:<br><br>Describe your Data that you are sending to the Monitoring here. <br><br>  <br><br>Technical Implementation is defined in ISD-4960755*** | CEC: Our UAT is for Testing for our Customers, meaning that they will add different Environments  to it – but not Prod. Our Prod env will receive Data from all ENV’s | Yes (please enter justification)  |              |                  |\n\n\n> [!TIP]\n> In case you experience a longer delay, please contact the department [here](fakeDepartmentEmail@ErsteGroup.Com)\n"
}

func TestReadMarkdownFileStructureRecursively(t *testing.T) {
	tests := []struct {
		name           string
		adapter        bitbucket.BitbucketApiServiceAdapter
		expectError    bool
		expectedResult []string
	}{
		{
			name:           "nilAdapter",
			adapter:        nil,
			expectError:    true,
			expectedResult: nil,
		},
		{
			name:           "nilResponse",
			adapter:        &MockBitbucketAdapter{},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name:           "noValuesContent",
			adapter:        &MockBitbucketAdapter{StreamFilesResponse: &bitbucketv1.APIResponse{}},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name:           "streamFilesReturnsError",
			adapter:        &MockBitbucketAdapter{StreamFilesResponse: nil, Error: errors.New("streamFiles")},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name: "valuesAreMissing",
			adapter: &MockBitbucketAdapter{StreamFilesResponse: &bitbucketv1.APIResponse{
				Values: map[string]any{
					"isLastPage":    true,
					"limit":         150,
					"nextPageStart": nil,
					"size":          15,
					"start":         0,
				},
			}},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name: "valuesIsNotAnySlice",
			adapter: &MockBitbucketAdapter{StreamFilesResponse: &bitbucketv1.APIResponse{
				Values: map[string]any{
					"values": 2,
				},
			}},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name: "valuesElementsAreNotString",
			adapter: &MockBitbucketAdapter{StreamFilesResponse: &bitbucketv1.APIResponse{
				Values: map[string]any{
					"values": []any{
						4356,
						1.89,
					},
				},
			}},
			expectError:    true,
			expectedResult: nil,
		},
		{
			name: "valuesIsNil",
			adapter: &MockBitbucketAdapter{StreamFilesResponse: &bitbucketv1.APIResponse{
				Values: map[string]any{
					"isLastPage":    true,
					"limit":         150,
					"nextPageStart": nil,
					"size":          15,
					"start":         0,
					"values":        nil,
				},
			}},
			expectError:    false,
			expectedResult: nil,
		},
		{
			name:        "valuesIsNil",
			adapter:     createMockBitbucketAdapter(),
			expectError: false,
			expectedResult: []string{
				"markdowns/Another-Folder/Crazy-Markdown.md",
				"markdowns/Another-Folder/What-a-Markdown-Example.md",
				"markdowns/Gateway/1-Onboarding.md",
				"markdowns/Gateway/2-Data-Preparation.md",
				"markdowns/Gateway/3-Visualization.md",
				"markdowns/GitHub-Flavored-Markdown/GitHub-Flavored-Markdown.md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := environment.Null()
			env.Logger = logging.DefaultLogger{Logger: zap.NewNop().Sugar()}

			reader := &bitbucket.O11yBitbucketReader{
				Env:     env,
				Adapter: tt.adapter,
			}

			gotFilePaths, err := reader.ReadMarkdownFileStructureRecursively("test_project", "test_repo", 0, 150)

			if tt.expectError {
				if err == nil {
					t.Fatalf("want error, but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("want NO error, but got: %v", err)
				}
			}

			if !cmp.Equal(gotFilePaths, tt.expectedResult) {
				t.Error(cmp.Diff(tt.expectedResult, gotFilePaths))
			}
		})
	}
}

func createMockBitbucketAdapter() *MockBitbucketAdapter {
	return &MockBitbucketAdapter{
		StreamFilesResponse: &bitbucketv1.APIResponse{
			Values: map[string]any{
				"isLastPage":    true,
				"limit":         150,
				"nextPageStart": nil,
				"size":          15,
				"start":         0,
				"values": []any{
					".gitignore",
					"images/gatewayImg1ObservabilityGrafanaRequest.png",
					"images/gatewayImg2AlloyConfigGen.png",
					"images/gatewayImg3GEMWebserver.png",
					"images/gatewayImg3GeorgeAT.png",
					"images/gatewayImg3Gevis.png",
					"images/gatewayImg3GrafanaDashboardFolder.png",
					"images/gatewayImg3Mainframe.png",
					"images/gatewayImg3Mercury.png",
					"markdowns/Another-Folder/Crazy-Markdown.md",
					"markdowns/Another-Folder/What-a-Markdown-Example.md",
					"markdowns/Gateway/1-Onboarding.md",
					"markdowns/Gateway/2-Data-Preparation.md",
					"markdowns/Gateway/3-Visualization.md",
					"markdowns/GitHub-Flavored-Markdown/GitHub-Flavored-Markdown.md",
				},
			},
		},
	}
}

func TestInitBitbucket(t *testing.T) {
	c := &config.Configuration{
		BitBucket: struct {
			Url         *config.JsonUrl
			User        string
			Password    string
			AccessToken string
			ProjectName string
			Repository  string
		}{
			//Url:         &config.JsonUrl{URL: &url.URL{Host: "api.bitbucket.org", Scheme: "https"}},
			User:        "your-username",
			Password:    "your-password",
			AccessToken: "your-access-token",
			ProjectName: "your-project",
			Repository:  "your-repository",
		},
	}

	env := environment.Null()
	env.Logger = logging.DefaultLogger{Logger: zap.NewNop().Sugar()}

	_, err := bitbucket.InitBitbucket(c, env)
	if err == nil {
		t.Fatal("want error, but got nil")
	}

	if !cmp.Equal("bitbucket url is not set", err.Error()) {
		t.Error(cmp.Diff("bitbucket url is not set", err.Error()))
		return
	}
}
