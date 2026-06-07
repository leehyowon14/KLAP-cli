# Lecture Plan Fixture

KLAS 강의계획서 조회 화면에서 확인한 API 흐름이다.

- Page: `GET /std/cps/atnlc/LectrePlanStdPage.do`
- Detail page: `POST /std/cps/atnlc/LectrePlanStdView.do`
- Verified sample: `2026,1` 컴퓨터그래픽스, 김동준

## List

강의계획서 목록 조회는 Vue 화면에서 아래 endpoint로 호출한다.

```http
POST /std/cps/atnlc/LectrePlanStdList.do
Content-Type: application/json;charset=UTF-8
```

최소 payload:

```json
{
  "selectSubj": "",
  "selectYear": "2026",
  "selecthakgi": "1",
  "selectYearHakgi": "2026,1",
  "selectRadio": "all",
  "selectText": "컴퓨터그래픽스",
  "selectProfsr": "",
  "cmmnGamok": "",
  "selecthakgwa": "",
  "selectMajor": ""
}
```

화면의 실제 Vue payload에는 빈 배열 필드도 같이 들어간다.

```json
{
  "list": [],
  "selectCmGamokList": [],
  "selectHwaGwakList": [],
  "selectMajorList": [],
  "selectYearList": []
}
```

컴퓨터그래픽스 목록 응답 샘플:

```json
[
  {
    "thisYear": "2026",
    "hakgi": "1",
    "openMajorCode": "I040",
    "openGrade": "3",
    "openGwamokNo": "3951",
    "bunbanNo": "01",
    "gwamokKname": "컴퓨터그래픽스",
    "memberName": "김동준",
    "codeName1": "전선",
    "sisuNum": 3,
    "hakjumNum": 3,
    "telNo": null,
    "summary": "본 과목은 컴퓨터 그래픽스의 핵심 이론과 실습을 연계하여 배우는 과정으로, Web 기반 3D 엔진을 실습 프레임워크로 활용한다.\r\n그래픽스 파이프라인, 좌표 변환, Rasterization, Shader 등 기본 개념을 익히고, Shading·Lighting·Texturing을 학습한 뒤\r\nGlobal Illumination, 물리 기반 렌더링(PBR), 애니메이션 시스템(Skeletal Animation, Keyframe, IK) 까지 확장한다.\r\n최종적으로 학생들은 실시간 3D 씬 + 동적 조명 + 캐릭터 애니메이션을 포함한 미니 엔진 수준의 결과물을 완성한다",
    "closeOpt": null,
    "videoUrl": null
  },
  {
    "thisYear": "2026",
    "hakgi": "1",
    "openMajorCode": "6120",
    "openGrade": "4",
    "openGwamokNo": "7083",
    "bunbanNo": "01",
    "gwamokKname": "컴퓨터그래픽스2",
    "memberName": "김종래",
    "codeName1": "전선",
    "sisuNum": 3,
    "hakjumNum": 3,
    "telNo": null,
    "summary": "3d 컴퓨터 그래픽의 기본 개념을 이해하고 작업 과정을 통한 결과물을 제작 과정을 학습한다.",
    "closeOpt": null,
    "videoUrl": null
  }
]
```

목록 행의 학정번호는 아래 필드 조합으로 표시된다.

```text
<openMajorCode>-<openGrade>-<openGwamokNo>-<bunbanNo>
```

예: `I040-3-3951-01`

## Detail Key

상세 조회 전 화면은 `selectSubj`를 생성한다.

```text
U<thisYear><hakgi><openGwamokNo><openMajorCode><bunbanNo><openGrade>
```

컴퓨터그래픽스 샘플:

```text
U202613951I040013
```

폐강 강의는 `closeOpt == "Y"`이면 상세 진입 전에 제외한다.
`summary == null`이면 화면에서는 "강의 계획서 정보가 없습니다!"를 띄우고 상세 진입을 막는다.

교양 고정 양식 여부는 아래 endpoint로 확인한다.

```http
POST /std/cps/atnlc/CultureOptOneInfo.do
Content-Type: application/json;charset=UTF-8
```

Payload:

```json
{
  "selectSubj": "U202613951I040013"
}
```

컴퓨터그래픽스 응답은 빈 body였다. 화면 로직은 `response.data.cultureOpt == null`이면 일반 상세 페이지를 사용한다.

## Detail Data

일반 강의계획서 상세 화면:

```http
POST /std/cps/atnlc/LectrePlanStdView.do
Content-Type: application/x-www-form-urlencoded
```

Form field:

```text
selectSubj=U202613951I040013
```

상세 화면은 로드 후 아래 JSON endpoint들을 호출한다.

```http
POST /std/cps/atnlc/LectrePlanData.do
POST /std/cps/atnlc/LectreEnginerGwamok.do
POST /std/cps/atnlc/LectrePlanInputTabFourgrid.do
POST /std/cps/atnlc/LectreBeforeGwamok.do
POST /std/cps/atnlc/LectrePlanInputTabSixInfo.do
POST /std/cps/atnlc/LectrePlanInputTabFiveInfo.do
POST /std/cps/atnlc/LectreTimeInfo.do
POST /std/cps/atnlc/LectreAstnt.do
POST /std/cps/atnlc/LectreTeam.do
```

공통 payload:

```json
{
  "selectSubj": "U202613951I040013"
}
```

`LectrePlanData.do` 주요 응답 필드:

```json
[
  {
    "openMajorCode": "I040",
    "openGrade": "3",
    "openGwamokNo": "3951",
    "bunbanNo": "01",
    "gwamokkename": "컴퓨터그래픽스-Computer Graphics",
    "openCode": "33",
    "codeName1": "전선",
    "hakjumNum": 3,
    "sisuNum": 3,
    "currentNum": "",
    "memberName": "김동준",
    "jikgeubName": "부교수",
    "telNo": null,
    "hpNo": null,
    "email": null,
    "gwamokKname": "컴퓨터그래픽스",
    "gwamokEname": "Computer Graphics",
    "summary": "본 과목은 컴퓨터 그래픽스의 핵심 이론과 실습을 연계하여 배우는 과정으로, Web 기반 3D 엔진을 실습 프레임워크로 활용한다.\r\n그래픽스 파이프라인, 좌표 변환, Rasterization, Shader 등 기본 개념을 익히고, Shading·Lighting·Texturing을 학습한 뒤\r\nGlobal Illumination, 물리 기반 렌더링(PBR), 애니메이션 시스템(Skeletal Animation, Keyframe, IK) 까지 확장한다.\r\n최종적으로 학생들은 실시간 3D 씬 + 동적 조명 + 캐릭터 애니메이션을 포함한 미니 엔진 수준의 결과물을 완성한다",
    "purpose": "그래픽스 파이프라인과 Shader 등 실시간 렌더링의 핵심 개념을 이해한다.\r\nGlobal Illumination과 물리 기반 조명 모델(PBR)을 구현하여 현실적인 조명 효과를 설계한다.\r\nKeyframe / Skeletal Animation을 통해 3D 객체와 캐릭터의 동적 움직임을 구현한다.",
    "result1": "그래픽스 파이프라인 이해\r\n물리 기반 조명 및 GI 구현\r\n애니메이션 시스템 구현\r\n그래픽스 시스템 통합 설계",
    "gwamokAble": "문제해결능력",
    "bookName": "강의교안 사용",
    "attendBiyul": 10,
    "learnBiyul": 10,
    "middleBiyul": 25,
    "lastBiyul": 25,
    "reportBiyul": 30,
    "face100Opt": "Y",
    "evaluationOpt": "evaluation_experiment",
    "week1Lecture": "오리엔테이션 + WebGL/Three.js 구조",
    "week2Lecture": "Transform Basics",
    "week3Lecture": "View / Projection / Camera",
    "week4Lecture": "Rasterization & Shader",
    "week5Lecture": "Shader 실습",
    "week6Lecture": "Texture Mapping",
    "week7Lecture": "PBR Material (BRDF, IBL)",
    "week8Lecture": "중간 시험",
    "week9Lecture": "Lighting Models (Phong → PBR)",
    "week10Lecture": "Global Illumination\r\n- Lightmap\r\n- Screen Space GI (SSAO/SSGI)\r\n- Ray tracing 개념",
    "week10Subs": "동영상 보강",
    "week11Lecture": "Skeletal Animation\r\n- Bone hierarchy\r\n- Skinning",
    "week11Subs": "동영상 보강",
    "week12Lecture": "Animation System\r\n- Keyframe interpolation\r\n- IK / Blend",
    "week13Lecture": "Acceleration + BVH + Ray query",
    "week14Lecture": "게임프로젝트 최종 Q/A",
    "week15Lecture": "기말 시험"
  }
]
```

`LectreTimeInfo.do` 응답 샘플:

```json
[
  {
    "dayname1": "화",
    "timeNo1": 3,
    "timeNo2": null,
    "timeNo3": null,
    "timeNo4": null,
    "locHname": "새빛103",
    "locCode": "300103",
    "locHname2": null,
    "locCode2": null,
    "code": "2         "
  },
  {
    "dayname1": "목",
    "timeNo1": 4,
    "timeNo2": null,
    "timeNo3": null,
    "timeNo4": null,
    "locHname": "새빛103",
    "locCode": "300103",
    "locHname2": null,
    "locCode2": null,
    "code": "4         "
  }
]
```

컴퓨터그래픽스 기준 빈 배열로 확인된 endpoint:

```text
LectreEnginerGwamok.do
LectrePlanInputTabFourgrid.do
LectreBeforeGwamok.do
LectrePlanInputTabSixInfo.do
LectreAstnt.do
LectreTeam.do
```

`LectrePlanInputTabFiveInfo.do`는 성과/설계 관련 확장 정보를 반환한다. 현재 CLI 구현에는 `LectrePlanData.do`와 `LectreTimeInfo.do`만으로 기본 강의계획서 출력이 가능하다.

## Parser Notes

- `selecthakgi`: `1`, `2`, `3`, `4`를 사용한다. `3`은 여름학기, `4`는 겨울학기다.
- 상세 데이터 대부분은 `LectrePlanData.do` 배열 첫 번째 원소에 있다.
- 주차별 강의는 `week1Lecture`부터 `week16Lecture`까지 key pattern으로 파싱한다.
- 보강/운영 방식은 `week1Subs`부터 `week16Subs`까지 key pattern으로 파싱한다.
- 평가 비율은 `attendBiyul`, `learnBiyul`, `middleBiyul`, `lastBiyul`, `reportBiyul`, `quizBiyul`, `gitaBiyul`이다.
- 대면/원격 운영 방식은 `face100Opt`, `faceliveOpt`, `live100Opt`, `facerecOpt`, `recliveOpt`, `rec100Opt`, `faceliverecOpt` 등 `Y` 여부로 판단한다.
